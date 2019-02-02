package web

import (
	"encoding/json"
	"github.com/filebrowser/filebrowser/src/config"
	fb "github.com/filebrowser/filebrowser/src/lib"
	"github.com/filebrowser/filebrowser/src/lib/fileutils"
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

func shareGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	//list of all shares
	if strings.HasPrefix(r.URL.Path, "/list") {
		sharesList := config.GetAllowedShares(c.User.Username, true)
		return renderJSON(w, sharesList)
	} else {
		item, uc := config.GetShare(c.User.Username, c.ShareUser, r.URL.Path)

		if item != nil && len(item.Path) > 0 {
			suffix := "?shareUser=" + c.ShareUser
			//replace user as for normal listing
			c.User = &fb.UserModel{uc, uc.Username, fileutils.Dir(uc.Scope), fileutils.Dir(uc.PreviewScope),}
			r.URL.Path = strings.Replace(r.URL.Path, "/"+uc.Username, "", 1)
			f, err := fb.GetInfo(r.URL, c)
			c.File = f

			if err != nil {
				return http.StatusNotFound, nil
			}
			err = f.GetListing(c.User, false)
			if err != nil {
				return http.StatusNotFound, nil
			}
			for _, itm := range f.Listing.Items {
				itm.URL += suffix
			}

			f.URL += suffix
			listingHandler(c, w, r)
			f.Listing.AllowGeneratePreview = c.Config.DefaultUser.AllowGeneratePreview

			return renderJSON(w, f)
		} else {
			return http.StatusNotFound, nil
		}
	}
	//

	return http.StatusNotFound, nil
}

func sharePostHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	itm := &config.ShareItem{}
	err := json.NewDecoder(r.Body).Decode(itm)
	if err != nil {
		return http.StatusBadRequest, err
	}
	itm.Path = r.URL.Path

	ok := c.User.AddShare(itm)
	if !ok {
		return http.StatusNotFound, nil
	}

	return http.StatusOK, nil
}

func shareDeleteHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {

	ok := c.User.DeleteShare(r.URL.Path)
	if !ok {
		return http.StatusNotFound, nil
	}

	return http.StatusOK, nil
}
