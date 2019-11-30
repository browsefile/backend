package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"net/http"
)

func settingsHandler(c *lib.Context) (int, error) {
	if c.URL != "" && c.URL != "/" {
		return http.StatusNotFound, nil
	}

	switch c.Method {
	case http.MethodGet:
		return settingsGetHandler(c)
	case http.MethodPut:
		return settingsPutHandler(c)
	}

	return http.StatusMethodNotAllowed, nil
}

func settingsGetHandler(c *lib.Context) (int, error) {
	if !c.User.Admin {
		return http.StatusForbidden, nil
	}
	return renderJSON(c, c.Config.CopyConfig())
}

func settingsPutHandler(c *lib.Context) (int, error) {
	if !c.User.Admin {
		return http.StatusForbidden, nil
	}

	// Checks if the request body is empty.
	if c.REQ.Body == nil {
		return http.StatusForbidden, cnst.ErrEmptyRequest
	}

	// Parses the request body and checks if it's well formed.
	mod := &config.GlobalConfig{}
	err := json.NewDecoder(c.REQ.Body).Decode(mod)
	if err != nil {
		return http.StatusBadRequest, err
	}
	mod.Verify()
	c.Config.UpdateConfig(mod)
	c.Config.RefreshUserRam()

	return http.StatusOK, nil
}
