package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/config"
	fb "github.com/browsefile/backend/src/lib"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

func shareHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	r.URL.Path = sanitizeURL(r.URL.Path)
	if c.User == nil && r.Method != "GET" {
		return http.StatusNotFound, nil
	}

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

func shareGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	//list of all shares
	var res = &fb.File{
		Listing: &fb.Listing{Items: make([]*fb.File, 0, 100)},
	}
	isDef := false
	isExternal := c.IsExternalShare()

	switch c.ShareType {
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
		item, uc := getShare(r.URL.Path, c)

		if item != nil && len(item.Path) > 0 {
			if !isExternal {
				item.Path = r.URL.Path
			}
			err, resLoc := shareListing(uc, item, c, w, r)

			if !checkShareErr(err, item.Path) {
				if !c.File.IsDir {
					res = c.File
					res.SetFileType(true)

					if res.Type == "text" {
						var content []byte
						//todo: fix me, what if file too big ?
						content, err = ioutil.ReadFile(res.Path)
						if err != nil {
							return ErrorToHTTP(err, true), err
						}

						res.Content = string(content)
					}
					// Tries to get the file type.

					isDef = true

					// Serve a preview if the file can't be edited or the
					// user has no permission to edit this file. Otherwise,
					// just serve the editor.
					if !res.CanBeEdited() || !c.User.AllowEdit {
						res.Kind = "preview"
					} else {
						res.Kind = "editor"
					}
					if isExternal {
						res.URL += "/?rootHash=" + c.RootHash
					}

				} else {
					if isExternal {
						for _, itm := range resLoc.Items {
							itm.URL = strings.Replace(itm.URL, item.Path, "", 1) + "&rootHash=" + c.RootHash
						}
					}
					merge(res.Listing, resLoc)
				}

				res.Name = c.File.Name
				res.Size = c.File.Size
				res.Language = c.File.Language

			}
			res.URL = r.URL.Path
			res.VirtualPath = item.Path
			res.Path = ""

		}
	}

	if !isDef && res.NumFiles == 0 && res.NumDirs == 0 {
		return http.StatusNotFound, nil
	}
	if isExternal {
		res.URL += "&share=" + c.ShareType
	} else {
		res.URL += "/?share=" + c.ShareType
	}

	if !isDef {
		res.IsDir = true
		res.VirtualPath = "/"
		res.Kind = "listing"
		// Copy the query values into the Listing struct
		if err := HandleSortOrder(c, w, r, "/"); err == nil {
			res.Sort = c.Sort
			res.Order = c.Order
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
	c.User = fb.ToUserModel(uc, c.Config)
	orig := r.URL.Path
	if c.IsExternalShare() {
		r.URL.Path = filepath.Join(shr.Path + r.URL.Path)
	} else {
		r.URL.Path = shr.Path
	}
	c.File, err = fb.MakeInfo(r.URL, c)
	if err != nil {
		log.Println(err)
	}
	if c.File.IsDir {
		if err != nil {
			return err, nil
		}

		listingHandler(c, w, r, nil)
		c.File.Listing.AllowGeneratePreview = c.Config.AllowGeneratePreview
		r.URL.Path = orig
		res = c.File.Listing
		suffix := "?share=" + uc.Username
		for _, itm := range res.Items {
			itm.URL += suffix
		}
	}

	return
}

func sharePostHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (res int, err error) {
	itm := &config.ShareItem{}
	if !strings.EqualFold(c.ShareType, "gen-ex") {
		err := json.NewDecoder(r.Body).Decode(itm)
		if strings.EqualFold(itm.Path, "") {
			return http.StatusBadRequest, err
		}
	}
	switch c.ShareType {
	case "gen-ex":
		shr := c.User.GetShare(r.URL.Path)
		var h string
		if shr == nil {
			h = config.GenShareHash(c.User.Username, r.URL.Path)
		} else {
			h = shr.Hash
		}

		l := c.Config.ExternalShareHost + "/shares?rootHash=" + url.QueryEscape(h)
		return renderJSON(w, l)

	case "my-meta":
		shr := c.User.GetShare(r.URL.Path)
		if shr != nil {
			if !c.User.DeleteShare(shr.Path) {
				return http.StatusBadRequest, err
			}
		}
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
