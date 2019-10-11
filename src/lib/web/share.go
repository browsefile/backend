package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"net/http"
	"net/url"
	"strings"
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
			return renderJSON(c.RESP, c.User.Shares)
		} else {
			shrs := c.User.GetShares(c.URL, false)
			var shr *config.ShareItem
			if len(shrs) == 0 {
				shr = &config.ShareItem{}
				shr.Path = c.URL
			} else {
				shr = shrs[0]
			}
			return renderJSON(c.RESP, shr)
		}

	default:
		return resourceGetHandler(c)
	}
}
func sharePostHandler(c *lib.Context) (res int, err error) {
	itm := &config.ShareItem{}
	if !strings.EqualFold(c.ShareType, "gen-ex") {
		err := json.NewDecoder(c.REQ.Body).Decode(itm)
		if strings.EqualFold(itm.Path, "") {
			return http.StatusBadRequest, err
		}
	}
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

		var h string
		if shrs == nil {
			h = config.GenShareHash(c.User.Username, c.URL)
		} else {
			h = shr.Hash
		}

		l := c.Config.ExternalShareHost + "/shares?rootHash=" + url.QueryEscape(h)
		return renderJSON(c.RESP, l)

	case "my-meta":
		shrs := c.User.GetShares(c.URL, false)
		var shr *config.ShareItem
		if len(shrs) == 0 {
			shr = &config.ShareItem{}
			shr.Path = c.URL
		} else {
			shr = shrs[0]
		}

		if shrs != nil {
			if !c.Config.DeleteShare(c.User.UserConfig, c.URL) {
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
		itm.Path = c.URL

		ok := c.User.AddShare(itm)
		if !ok {
			return http.StatusNotFound, nil
		}
	}
	c.Config.Update(c.User.UserConfig)
	return renderJSON(c.RESP, itm)
}

func shareDeleteHandler(c *lib.Context) (int, error) {
	ok := c.Config.DeleteShare(c.User.UserConfig, c.URL)
	if !ok {
		return http.StatusNotFound, nil
	}
	c.Config.Update(c.User.UserConfig)
	return http.StatusOK, nil
}
