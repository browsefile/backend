package web

import (
	"context"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib"
	"net/http"
	"strings"
)

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func ServeDav(c *lib.Context, w http.ResponseWriter, r *http.Request) {
	if !authDavHandler(c, w, r) {
		return
	}
	if r.Method == "HEAD" {
		w = newResponseWriterNoBody(w)
	}

	// If this request modified the files and the user doesn't have permission, or modify any share
	// to do so, return forbidden.
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "MKCOL" ||
		r.Method == "DELETE" || r.Method == "COPY" || r.Method == "MOVE" {
		if (strings.HasPrefix(r.URL.Path, cnst.WEB_DAV_URL+"/shares") ||
			!strings.HasPrefix(r.URL.Path, cnst.WEB_DAV_URL+"/files")) ||
			!(c.User.AllowEdit || c.User.AllowNew) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	// Excerpt from RFC4918, section 9.4:
	//
	// 		GET, when applied to a collection, may return the contents of an
	//		"index.html" resource, a human-readable view of the contents of
	//		the collection, or something else altogether.
	//
	// Get, when applied to collection, will return the same as PROPFIND method.
	if r.Method == "GET" {
		info, err := c.User.DavHandler.FileSystem.Stat(context.TODO(), r.URL.Path)
		if err == nil && info.IsDir() {
			r.Method = "PROPFIND"

			if r.Header.Get("Depth") == "" {
				r.Header.Add("Depth", "1")
			}
		}
	}

	// Runs the WebDAV.
	c.User.DavHandler.ServeHTTP(w, r)
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
