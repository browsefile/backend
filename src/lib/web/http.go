package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/errors"
	fb "github.com/browsefile/backend/src/lib"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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

		code, err := serve(c, w, r)

		if code >= 400 {
			txt := http.StatusText(code)
			if len(c.Params.PreviewType) == 0 {
				log.Printf("%v: %v %v\n", r.URL.Path, code, txt)
			} else {
				err = nil
			}
		}

		if err != nil {
			log.Print(err)
		}
	})
}

// serve is the main entry point of this HTML application.
func serve(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// Checks if this request is made to the static assets folder. If so, and
	// if it is a GET request, returns with the asset. Otherwise, returns
	// a status not implemented.
	if matchURL(r.URL.Path, "/static") {
		if r.Method != http.MethodGet {
			return http.StatusNotImplemented, nil
		}

		return staticHandler(c, w, r)
	}

	// Checks if this request is made to the API and directs to the
	// API handler if so.
	if matchURL(r.URL.Path, "/api") {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		return apiHandler(c, w, r)
	}

	// Any other request should show the index.html file.
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("x-xss-protection", "1; mode=block")

	return renderFile(c, r, w, "index.html")
}

// staticHandler handles the static assets path.
func staticHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (code int, err error) {
	if r.URL.Path != "/static/manifest.json" {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/static")
		http.FileServer(c.Assets.HTTPBox()).ServeHTTP(w, r)
		return 0, nil
	}

	return renderFile(c, r, w, "manifest.json")
}

// apiHandler is the main entry point for the /api endpoint.
func apiHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (code int, err error) {
	if r.URL.Path == "/auth/get" {
		return authHandler(c, w, r)
	}

	if r.URL.Path == "/auth/renew" {
		return renewAuthHandler(c, w, r)
	}

	valid, _ := validateAuth(c, r)
	if !valid {
		return http.StatusForbidden, nil
	}
	isShares := ProcessParams(c, r)
	//allow only GET requests, for external share
	if valid && c.User.IsGuest() && (!isShares ||
		!strings.EqualFold(r.Method, http.MethodGet) ||
		strings.HasPrefix(r.URL.Path, "/resource") ||
		strings.HasPrefix(r.URL.Path, "/user")||
		strings.HasPrefix(r.URL.Path, "/sett")) {
		return http.StatusForbidden, nil
	}

	if c.Router == "download" {
		c.File, err = fb.MakeInfo(r.URL, c)
		c.File.SetFileType(false)
		m := mime.TypeByExtension(c.File.Extension)
		if len(m) == 0 {
			m = c.File.Type
		}
		w.Header().Set("Content-Type", m)

		if err != nil {
			return ErrorToHTTP(err, false), err
		}
	}

	if c.Checksum != "" {
		err := c.File.Checksum(c.Checksum)
		if err == errors.ErrInvalidOption {
			return http.StatusBadRequest, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
		// do not waste bandwidth if we just want the checksum
		c.File.Content = ""
		return renderJSON(w, c.File)
	}

	switch c.Router {
	case "download":
		code, err = downloadHandler(c, w, r)
	case "download-share":
		code, err = downloadSharesHandler(c, w, r)
	case "resource":
		code, err = resourceHandler(c, w, r)
	case "users":
		code, err = usersHandler(c, w, r)
	case "settings":
		code, err = settingsHandler(c, w, r)
	case "shares":
		code, err = shareHandler(c, w, r)
	case "search":
		code, err = searchHandler(c, w, r)
	case "playlist":
		code, err = makePlaylist(c, w, r)
	case "playlist-share":
		code, err = makeSharePlaylist(c, w, r)

	default:
		code = http.StatusNotFound
	}

	return code, err
}

// renderFile renders a file using a template with some needed variables.
func renderFile(c *fb.Context, r *http.Request, w http.ResponseWriter, file string) (int, error) {
	contentType := mime.TypeByExtension(filepath.Ext(file))
	if len(contentType) == 0 {
		contentType = "text"
	}
	c.Query = r.URL.Query()

	c.RootHash = c.Query.Get("rootHash")
	isEx := len(c.RootHash) > 0
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	data := map[string]interface{}{
		"Name":            "Browsefile",
		"DisableExternal": false,
		"Version":         fb.Version,
		"isExternal":      isEx,
		"StaticURL":       "/static",
		"Signup":          false,
		"NoAuth":          strings.ToLower(c.Config.Method) == "noauth" || strings.ToLower(c.Config.Method) == "ip",
		"ReCaptcha":       c.ReCaptcha.Key != "" && c.ReCaptcha.Secret != "",
		"ReCaptchaHost":   c.ReCaptcha.Host,
		"ReCaptchaKey":    c.ReCaptcha.Key,
	}

	if isEx {
		data["StaticURL"] = c.Config.ExternalShareHost + "/static"
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	data["Json"] = template.JS(string(b))

	index := template.Must(template.New("index").Delims("[{[", "]}]").Parse(c.Assets.MustString(file)))

	err = index.Execute(w, data)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}

// renderJSON prints the JSON version of data to the browser.
func renderJSON(w http.ResponseWriter, data interface{}) (int, error) {
	marsh, err := json.Marshal(data)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(marsh); err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil
}

// matchURL checks if the first URL matches the second.
func matchURL(first, second string) bool {
	first = strings.ToLower(first)
	second = strings.ToLower(second)

	return strings.HasPrefix(first, second)
}

// ErrorToHTTP converts errors to HTTP Status Code.
func ErrorToHTTP(err error, gone bool) int {
	switch {
	case err == nil:
		return http.StatusOK
	case os.IsPermission(err):
		return http.StatusForbidden
	case os.IsNotExist(err):
		if !gone {
			return http.StatusNotFound
		}

		return http.StatusGone
	case os.IsExist(err):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
