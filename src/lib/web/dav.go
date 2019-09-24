package web

import (
	"context"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"os"
	"strings"
)

type DavFsDelegate struct {
	webdav.Dir
}

func trimWebDav(p string) string {
	return strings.TrimPrefix(p, cnst.WEB_DAV_URL)
}
func (d DavFsDelegate) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return d.Dir.Mkdir(ctx, trimWebDav(name), perm)
}

func (d DavFsDelegate) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return d.Dir.OpenFile(ctx, trimWebDav(name), flag, perm)
}
func (d DavFsDelegate) RemoveAll(ctx context.Context, name string) error {
	return d.Dir.RemoveAll(ctx, trimWebDav(name))
}

func (d DavFsDelegate) Rename(ctx context.Context, oldName, newName string) error {
	return d.Dir.Rename(ctx, trimWebDav(oldName), trimWebDav(newName))
}

func (d DavFsDelegate) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return d.Dir.Stat(ctx, trimWebDav(name))
}

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func ServeDav(c *lib.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	// Gets the correct user for this request.
	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Not authorized", 401)
		return
	}

	user, _ := c.Config.GetByUsername("admin")
	if !ok {
		http.Error(w, "Not authorized", 401)
		return
	}

	if !lib.CheckPasswordHash(password, user.Password) {
		log.Println("Wrong Password for user", username)
		http.Error(w, "Not authorized", 401)
		return
	}
	if r.Method == "HEAD" {
		w = newResponseWriterNoBody(w)
	}

	// If this request modified the files and the user doesn't have permission
	// to do so, return forbidden.
	if (r.Method == "PUT" || r.Method == "POST" || r.Method == "MKCOL" ||
		r.Method == "DELETE" || r.Method == "COPY" || r.Method == "MOVE") &&
		!(user.AllowEdit || user.AllowNew) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Excerpt from RFC4918, section 9.4:
	//
	// 		GET, when applied to a collection, may return the contents of an
	//		"index.html" resource, a human-readable view of the contents of
	//		the collection, or something else altogether.
	//
	// Get, when applied to collection, will return the same as PROPFIND method.
	if r.Method == "GET" {
		info, err := user.DavHandler.FileSystem.Stat(context.TODO(), r.URL.Path)
		if err == nil && info.IsDir() {
			r.Method = "PROPFIND"

			if r.Header.Get("Depth") == "" {
				r.Header.Add("Depth", "1")
			}
		}
	}

	// Runs the WebDAV.
	user.DavHandler.ServeHTTP(w, r)
}

// responseWriterNoBody is a wrapper used to suprress the body of the response
// to a request. Mainly used for HEAD requests.
type responseWriterNoBody struct {
	http.ResponseWriter
}

// newResponseWriterNoBody creates a new responseWriterNoBody.
func newResponseWriterNoBody(w http.ResponseWriter) *responseWriterNoBody {
	return &responseWriterNoBody{w}
}

// Header executes the Header method from the http.ResponseWriter.
func (w responseWriterNoBody) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write suprresses the body.
func (w responseWriterNoBody) Write(data []byte) (int, error) {
	return 0, nil
}

// WriteHeader writes the header to the http.ResponseWriter.
func (w responseWriterNoBody) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}
