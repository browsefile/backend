package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/config"
	fb "github.com/browsefile/backend/src/lib"
	"net/http"
	"net/url"
	"strings"
)

func shareHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	r.URL.Path = sanitizeURL(r.URL.Path)
	if c.User == nil && r.Method != "GET" {
		return http.StatusNotFound, nil
	}

	switch r.Method {
	case http.MethodGet:
		return shareGetHandler(c, w, r, nil)
	case http.MethodDelete:
		return shareDeleteHandler(c, w, r)
	case http.MethodPost:
		return sharePostHandler(c, w, r)
	}

	return http.StatusNotImplemented, nil
}

func shareGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request, fitFilter fb.FitFilter) (int, error) {
	switch c.ShareType {
	case "my-meta":
		if "/" == r.URL.Path {
			return renderJSON(w, c.User.Shares)
		} else {
			shr := c.User.GetShare(r.URL.Path, false)
			if shr == nil {
				shr = &config.ShareItem{}
				shr.Path = r.URL.Path
			}
			return renderJSON(w, shr)
		}

	default:
		return resourceGetHandler(c, w, r, fitFilter)
	}
} /*
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

func shareListing(uc *config.UserConfig, shr *config.ShareItem, c *fb.Context, w http.ResponseWriter, r *http.Request, fitFilter fb.FitFilter) (err error, res *fb.Listing) {
	//replace user as for normal listing
	c.User = fb.ToUserModel(uc, c.Config)
	orig := r.URL.Path
	isExternal := c.IsExternalShare()
	if isExternal {
		r.URL.Path = filepath.Join(shr.Path, r.URL.Path)
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

		listingHandler(c, w, r, fitFilter)
		c.File.Listing.AllowGeneratePreview = len(c.Config.ScriptPath) > 0
		r.URL.Path = orig
		res = c.File.Listing
		if !isExternal {
			suffix := "?share=" + uc.Username
			for _, itm := range res.Items {
				itm.URL += suffix
			}
		}

	}

	return
}
*/
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
		shr := c.User.GetShare(r.URL.Path, false)
		var h string
		if shr == nil {
			h = config.GenShareHash(c.User.Username, r.URL.Path)
		} else {
			h = shr.Hash
		}

		l := c.Config.ExternalShareHost + "/shares?rootHash=" + url.QueryEscape(h)
		return renderJSON(w, l)

	case "my-meta":
		shr := c.User.GetShare(r.URL.Path, false)
		if shr != nil {
			if !c.Config.DeleteShare(c.User.UserConfig, r.URL.Path) {
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

	ok := c.Config.DeleteShare(c.User.UserConfig, r.URL.Path)
	if !ok {
		return http.StatusNotFound, nil
	}
	c.Config.Update(c.User.UserConfig)
	return http.StatusOK, nil
}
