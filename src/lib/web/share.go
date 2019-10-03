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
