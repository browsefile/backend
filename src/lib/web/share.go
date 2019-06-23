package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/config"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"log"
	"net/http"
	"strings"
)

func shareHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	r.URL.Path = sanitizeURL(r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		return shareGetHandler(c, w, r)
	case http.MethodDelete:
		return shareDeleteHandler(c, w, r)
	case http.MethodPost:
		return sharePostHandler(c, w, r)
	}

	return http.StatusNotImplemented, nil
}
func isShareHandler(share string) bool {
	return strings.EqualFold(share, "my-list") || strings.EqualFold(share, "my") || strings.EqualFold(share, "list")
}

func shareGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	//list of all shares
	var res = &fb.File{
		Listing: &fb.Listing{Items: make([]*fb.File, 0, 100)},
	}
	isDef := false
	switch c.ShareUser {
	case "my-list":
		for _, shr := range c.User.Shares {
			err, resLoc := shareListing(c.User.UserConfig, shr, c, w, r)
			if !checkShareErr(err, shr.Path) {
				merge(res.Listing, resLoc)
			}
		}
	case "my":
		shr := c.User.GetShare(r.URL.Path)
		err, item := shareListing(c.User.UserConfig, shr, c, w, r)
		if !checkShareErr(err, shr.Path) {
			merge(res.Listing, item)
		}
	case "my-meta":
		if "/" == r.URL.Path {
			return renderJSON(w, c.User.Shares)
		} else {
			shr := c.User.GetShare(r.URL.Path)
			if shr == nil {
				shr = &config.ShareItem{}
				shr.Path = r.URL.Path
			}
			return renderJSON(w, shr)
		}

	case "list":
		for _, v := range config.GetAllowedShares(c.User.Username, true) {
			for _, item := range v {
				err, resLoc := shareListing(item.UserConfig, item.ShareItem, c, w, r)
				if !checkShareErr(err, item.Path) {
					merge(res.Listing, resLoc)
				}
			}
		}
	default:
		//share resource handler
		item, uc := config.GetShare(c.User.Username, c.ShareUser, r.URL.Path)
		if item != nil && len(item.Path) > 0 {
			item.Path = r.URL.Path
			err, resLoc := shareListing(uc, item, c, w, r)

			if !checkShareErr(err, item.Path) {
				if !c.File.IsDir {
					res = c.File
					res.SetFileType(false)
					// Tries to get the file type.
					if err = res.SetFileType(true); err != nil {
						return ErrorToHTTP(err, true), err
					}
					isDef = true

					// Serve a preview if the file can't be edited or the
					// user has no permission to edit this file. Otherwise,
					// just serve the editor.
					if !res.CanBeEdited() || !c.User.AllowEdit {
						res.Kind = "preview"
					}

					res.Kind = "editor"

				} else {
					merge(res.Listing, resLoc)
					res.IsDir = c.File.IsDir
				}
			}
			res.URL = r.URL.Path
			res.VirtualPath = item.Path
			res.Path = ""

		}
	}

	if !isDef && res.NumFiles == 0 && res.NumDirs == 0 {
		return http.StatusNotFound, nil
	}
	res.URL = "/?share=" + c.ShareUser
	if !isDef {
		res.IsDir = true
		res.VirtualPath = "/"
		res.Kind = "listing"
		cookieScope := c.RootURL()
		if cookieScope == "" {
			cookieScope = "/"
		}
		// Copy the query values into the Listing struct
		if sort, order, err := HandleSortOrder(w, r, cookieScope); err == nil {
			res.Sort = sort
			res.Order = order
			res.ApplySort()
		}

		res.AllowGeneratePreview = c.Config.AllowGeneratePreview
	}

	return renderJSON(w, res)
}
func checkShareErr(err error, path string) (res bool) {
	if err != nil {
		log.Printf("cant fetch share path %v", path)
		res = true
	}
	return
}
func merge(fin, n *fb.Listing) {
	fin.Items = append(fin.Items, n.Items...)
	fin.NumDirs += n.NumDirs
	fin.NumFiles += n.NumFiles
}

func shareListing(uc *config.UserConfig, shr *config.ShareItem, c *fb.Context, w http.ResponseWriter, r *http.Request) (err error, res *fb.Listing) {
	//replace user as for normal listing
	c.User = &fb.UserModel{uc, uc.Username, fileutils.Dir(uc.Scope), fileutils.Dir(uc.PreviewScope)}
	orig := r.URL.Path

	r.URL.Path = shr.Path
	c.File, err = fb.GetInfo(r.URL, c)
	if err != nil {
		log.Println(err)
	}
	if c.File.IsDir {
		if err != nil {
			return err, nil
		}
		err = c.File.GetListing(c.User, false)
		if err != nil {
			return err, nil
		}

		listingHandler(c, w, r)
		c.File.Listing.AllowGeneratePreview = c.Config.AllowGeneratePreview
		r.URL.Path = orig
		res = c.File.Listing
		suffix := "?share=" + uc.Username
		for _, itm := range res.Items {
			itm.URL += suffix
			itm.URL = strings.Replace(itm.URL, "/files", "/shares", 1)
			itm.Path = ""
		}
	}

	return
}

func sharePostHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	itm := &config.ShareItem{}
	err := json.NewDecoder(r.Body).Decode(itm)
	if strings.EqualFold(itm.Path, "") {
		return http.StatusBadRequest, err
	}
	switch c.ShareUser {
	case "my-meta":
		shr := c.User.GetShare(r.URL.Path)
		if shr != nil {
			if !c.User.DeleteShare(shr.Path) {
				return http.StatusBadRequest, err
			}
		}
		itm.Path = strings.Replace(itm.Path, "/files", "", 1)
		if !c.User.AddShare(itm) {
			return http.StatusBadRequest, err
		}
	default:

		if err != nil {
			return http.StatusBadRequest, err
		}
		itm.Path = r.URL.Path

		ok := c.User.AddShare(itm)
		if !ok {
			return http.StatusNotFound, nil
		}
	}
	c.Config.Update(c.User.UserConfig)
	return renderJSON(w, itm)
}

func shareDeleteHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {

	ok := c.User.DeleteShare(r.URL.Path)
	if !ok {
		return http.StatusNotFound, nil
	}
	c.Config.Update(c.User.UserConfig)
	return http.StatusOK, nil
}
