package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"html/template"
	"log"
	"net/http"

	"strings"
)

// Handler returns a function compatible with web.HandleFunc.
func Handler(m *fb.FileBrowser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := &fb.Context{
			FileBrowser: m,
			User:        nil,
			File:        nil,
			Params:      new(fb.Params),
		}
		c.REQ = r
		c.RESP = w
		c.Method = c.REQ.Method
		code, err := serve(c)

		if err != nil {
			txt := http.StatusText(code)
			if len(c.Params.PreviewType) == 0 {
				log.Printf("%v: %v %v\n", r.URL.Path, code, txt)
				log.Println(err)
			} else {
				log.Println(err)
				err = nil
			}
		}
		if !c.Rendered && c.Router > 0 && c.Router != cnst.R_DOWNLOAD && c.Router != cnst.R_PLAYLIST {
			w.WriteHeader(code)
		}

	})
}

// serve is the main entry point of this HTML application.
func serve(c *fb.Context) (int, error) {
	// Checks if this request is made to the static assets folder. If so, and
	// if it is a GET request, returns with the asset. Otherwise, returns
	// a status not implemented.
	if strings.HasPrefix(c.REQ.URL.Path, "/static") {
		if c.Method != http.MethodGet {
			return http.StatusNotImplemented, nil
		}

		return staticHandler(c)
	}
	if strings.HasPrefix(c.REQ.URL.Path, cnst.WEB_DAV_URL) {
		ServeDav(c, c.RESP, c.REQ)
		return http.StatusOK, nil

	}

	// Checks if this request is made to the API and directs to the
	// API handler if so.
	if strings.HasPrefix(c.REQ.URL.Path, "/api") {
		return apiHandler(c)
	}

	// Any other request should show the index.html file.
	c.RESP.Header().Set("x-content-type-options", "nosniff")
	c.RESP.Header().Set("x-xss-protection", "1; mode=block")

	return renderFile(c, "index.html")
}

// staticHandler handles the static assets path.
func staticHandler(c *fb.Context) (code int, err error) {
	if c.REQ.URL.Path != "/static/manifest.json" {
		c.REQ.URL.Path = strings.TrimPrefix(c.REQ.URL.Path, "/static")
		http.FileServer(c.Assets.HTTPBox()).ServeHTTP(c.RESP, c.REQ)
		return 0, nil
	}

	return renderFile(c, "manifest.json")
}

// apiHandler is the main entry point for the /api endpoint.
func apiHandler(c *fb.Context) (code int, err error) {
	c.REQ.URL.Path = strings.TrimPrefix(c.REQ.URL.Path, "/api")
	if c.REQ.URL.Path == "/auth/get" {
		return authHandler(c)
	}

	if c.REQ.URL.Path == "/auth/renew" {
		return renewAuthHandler(c)
	}
	valid, _ := validateAuth(c)

	if !valid {
		return http.StatusForbidden, nil
	}
	isShares := ProcessParams(c)
	//allow only GET requests, for external share
	if valid && c.User.IsGuest() && (!isShares ||
		c.Method != http.MethodGet ||
		c.Router == cnst.R_USERS ||
		c.Router == cnst.R_SETTINGS) {
		return http.StatusForbidden, nil
	}

	if c.Checksum != "" {
		err := c.File.Checksum(c.Checksum)
		if err == cnst.ErrInvalidOption {
			return http.StatusBadRequest, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
		// do not waste bandwidth if we just want the checksum
		c.File.Content = ""
		return renderJSON(c, c.File)
	}

	switch c.Router {
	case cnst.R_DOWNLOAD:
		code, err = downloadHandler(c)
	case cnst.R_RESOURCE:
		code, err = resourceHandler(c)
	case cnst.R_USERS:
		code, err = usersHandler(c)
	case cnst.R_SETTINGS:
		code, err = settingsHandler(c)
	case cnst.R_SHARES:
		code, err = shareHandler(c)
	case cnst.R_SEARCH:
		code, err = searchHandler(c)
	case cnst.R_PLAYLIST:
		code, err = makePlaylist(c)

	default:
		code = http.StatusNotFound
	}
	if (c.Router == cnst.R_SETTINGS ||
		c.Router == cnst.R_USERS ||
		c.Router == cnst.R_RESOURCE ||
		c.Router == cnst.R_SHARES) &&
		c.Method == http.MethodPatch ||
		c.Method == http.MethodPut ||
		c.Method == http.MethodPost ||
		c.Method == http.MethodDelete {
		if c.Config != nil && err != nil {
			c.Config.WriteConfig()
		}

	}

	return code, err
}

// renderFile renders a file using a template with some needed variables.
func renderFile(c *fb.Context, file string) (int, error) {
	contentType := utils.GetMimeType(file)
	if len(contentType) == 0 {
		contentType = cnst.TEXT
	}
	c.Query = c.REQ.URL.Query()

	c.IsExternal = len(c.Query.Get(cnst.P_EXSHARE)) > 0
	c.RESP.Header().Set("Content-Type", contentType+"; charset=utf-8")
	cfgM := c.GetAuthConfig()

	data := map[string]interface{}{
		"Name":            "Browsefile",
		"DisableExternal": false,
		"Version":         cnst.Version,
		"isExternal":      c.IsExternal,
		"StaticURL":       "/static",
		"Signup":          false,
		"NoAuth":          strings.ToLower(cfgM.AuthMethod) == "noauth" || strings.ToLower(cfgM.AuthMethod) == "ip",
		"ReCaptcha":       c.ReCaptcha.Key != "" && c.ReCaptcha.Secret != "",
		"ReCaptchaHost":   c.ReCaptcha.Host,
		"ReCaptchaKey":    c.ReCaptcha.Key,
	}

	if c.IsExternal {
		data["StaticURL"] = c.Config.ExternalShareHost + "/static"
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	data["Json"] = template.JS(string(b))

	index := template.Must(template.New("index").Delims("[{[", "]}]").Parse(c.Assets.MustString(file)))

	err = index.Execute(c.RESP, data)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}

// renderJSON prints the JSON version of data to the browser.
func renderJSON(c *fb.Context, data interface{}) (int, error) {
	marsh, err := json.Marshal(data)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	c.Rendered = true
	c.RESP.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := c.RESP.Write(marsh); err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}
