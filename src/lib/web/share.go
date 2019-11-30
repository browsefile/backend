package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"log"
	"net/http"
)

func shareHandler(c *lib.Context) (int, error) {
	c.URL = sanitizeURL(c.URL)
	if c.User == nil && c.Method != "GET" {
		return http.StatusNotFound, nil
	}
	switch c.Method {
	case http.MethodGet:
		c.FitFilter = func(name, p string) bool {
			return resourceMediaFilter(c, name, p)
		}
		return shareGetHandler(c)
	case http.MethodDelete:
		return shareDeleteHandler(c)
	case http.MethodPost:
		return sharePostHandler(c)
	}

	return http.StatusNotImplemented, nil
}

func shareGetHandler(c *lib.Context) (int, error) {
	switch c.ShareType {
	case "my-meta":
		if "/" == c.URL {
			return renderJSON(c, c.User.Shares)
		} else {
			shrs := c.User.GetShares(c.URL, false)
			var shr *config.ShareItem
			if len(shrs) == 0 {
				shr = &config.ShareItem{}
				shr.Path = c.URL
			} else {
				shr = shrs[0]
			}
			return renderJSON(c, shr)
		}

	default:
		return resourceGetHandler(c)
	}
}
func sharePostHandler(c *lib.Context) (res int, err error) {
	itm := &config.ShareItem{}
	if c.ShareType != "gen-ex" {
		err := json.NewDecoder(c.REQ.Body).Decode(itm)
		if itm.Path == "" {
			return http.StatusBadRequest, err
		}
	}
	needUpd := false
	switch c.ShareType {
	case "gen-ex":
		shrs := c.User.GetShares(c.URL, false)
		var shr *config.ShareItem
		if len(shrs) == 0 {
			shr = &config.ShareItem{}
			shr.Path = c.URL
		} else {
			shr = shrs[0]
		}

		l := c.Config.ExternalShareHost + "/shares/" + shr.ResolveSymlinkName() + "?" + cnst.P_EXSHARE + "=1"
		return renderJSON(c, l)

	default:
		shrs := c.User.GetShares(itm.Path, false)
		if shrs != nil && !c.User.DeleteShare(itm.Path) {
			return http.StatusBadRequest, cnst.ErrExist
		}
		if !c.User.AddShare(itm) {
			return http.StatusBadRequest, cnst.ErrExist
		}
		needUpd = true
	}
	if needUpd {
		if err = c.Config.Update(c.User.UserConfig); err != nil {
			log.Println(err)
			return http.StatusBadRequest, err
		}
	}
	return renderJSON(c, itm)
}

func shareDeleteHandler(c *lib.Context) (int, error) {
	if !c.User.DeleteShare(c.URL) {
		return http.StatusNotFound, nil
	} else {
		if err := c.Config.Update(c.User.UserConfig); err != nil {
			log.Println(err)
			return http.StatusNotFound, nil
		}

	}

	return http.StatusOK, nil
}
