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
	"strconv"
	"strings"
)

// Handler returns a function compatible with web.HandleFunc.
func Handler(m *fb.FileBrowser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := &fb.Context{
			FileBrowser: m,
			User:        nil,
			File:        nil,
		}

		code, err := serve(c, w, r)

		if code >= 400 {
			txt := http.StatusText(code)
			if len(c.PreviewType) == 0 {
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

	/*	if strings.HasPrefix(r.URL.Path, "/share/") {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/share/")
		return sharePage(c, w, r)
	}*/

	// Any other request should show the index.html file.
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("x-xss-protection", "1; mode=block")

	return renderFile(c, w, "index.html")
}

// staticHandler handles the static assets path.
func staticHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path != "/static/manifest.json" {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/static")
		http.FileServer(c.Assets.HTTPBox()).ServeHTTP(w, r)
		return 0, nil
	}

	return renderFile(c, w, "manifest.json")
}

// apiHandler is the main entry point for the /api endpoint.
func apiHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
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

	c.Router, r.URL.Path = splitURL(r.URL.Path)
	isShares := strings.HasPrefix(c.Router, "shares")

	processParams(c, r)

	//redirect to the real handler in shares case
	if isShares {
		//possibility to process shares view/download
		if !isShareHandler(c.ShareUser) {
			c.Router, r.URL.Path = splitURL(r.URL.Path)
		}

		if c.Router == "download" {
			c.Router = "download-share"
		} else if c.Router == "resource" {
			c.Router = "shares"
		}
	}

	checksum := r.URL.Query().Get("checksum")
	if c.Router == "download" || c.Router == "subtitle" || c.Router == "subtitles" {
		var err error
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
	if checksum != "" {
		err := c.File.Checksum(checksum)
		if err == errors.ErrInvalidOption {
			return http.StatusBadRequest, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
		// do not waste bandwidth if we just want the checksum
		c.File.Content = ""
		return renderJSON(w, c.File)
	}
	var code int
	var err error
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
	case "subtitles":
		code, err = subtitlesHandler(c, w, r)
	case "subtitle":
		code, err = subtitleHandler(c, w, r)
	case "search":
		code, err = searchHandler(c, w, r)
	default:
		code = http.StatusNotFound
	}

	return code, err
}

func processParams(c *fb.Context, r *http.Request) {
	queryValues := r.URL.Query()
	c.PreviewType = queryValues.Get("previewType")
	c.ShareUser = queryValues.Get("share")

	q := r.URL.Query().Get("query")
	if len(q) > 0 {
		if strings.Contains(q, "type") {
			arr := strings.Split(q, ":")
			arr = strings.Split(arr[1], " ")
			c.SearchString = arr[1]
			c.SearchType = arr[0]
		} else {
			c.SearchString = q
		}
		queryValues.Del("query")
		r.URL.RawQuery = queryValues.Encode()
	}

	if len(c.ShareUser) > 0 {
		queryValues.Del("share")
		r.URL.RawQuery = queryValues.Encode()
	}
	c.IsRecursive, _ = strconv.ParseBool(queryValues.Get("recursive"))
	if c.IsRecursive {
		queryValues.Del("recursive")
		r.URL.RawQuery = queryValues.Encode()
	}

}

// splitURL splits the path and returns everything that stands
// before the first slash and everything that goes after.
func splitURL(path string) (string, string) {
	if path == "" {
		return "", ""
	}

	path = strings.TrimPrefix(path, "/")

	i := strings.Index(path, "/")
	if i == -1 {
		return "", path
	}

	return path[0:i], path[i:]
}

// renderFile renders a file using a template with some needed variables.
func renderFile(c *fb.Context, w http.ResponseWriter, file string) (int, error) {
	var contentType string
	switch filepath.Ext(file) {
	case ".html":
		contentType = "text/html"
	case ".js":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".css":
		contentType = "text/css"

	default:
		contentType = "text"
	}

	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	data := map[string]interface{}{
		"Name":            "Browsefile",
		"DisableExternal": false,
		"Version":         fb.Version,
		"StaticURL":       strings.TrimPrefix("/static", "/"),
		"Signup":          true,
		"NoAuth":          strings.ToLower(c.Config.Method) == "noauth" || strings.ToLower(c.Config.Method) == "ip",
		"ReCaptcha":       c.ReCaptcha.Key != "" && c.ReCaptcha.Secret != "",
		"ReCaptchaHost":   c.ReCaptcha.Host,
		"ReCaptchaKey":    c.ReCaptcha.Key,
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
