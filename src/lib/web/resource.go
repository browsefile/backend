package web

import (
	"errors"
	"fmt"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// sanitizeURL sanitizes the URL to prevent path transversal
// using fileutils.SlashClean and adds the trailing slash bar.
func sanitizeURL(url string) string {
	path := fileutils.SlashClean(url)
	if strings.HasSuffix(url, "/") && path != "/" {
		return path + "/"
	}
	return path
}

func resourceHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	r.URL.Path = sanitizeURL(r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		return resourceGetHandler(c, w, r)
	case http.MethodDelete:
		return resourceDeleteHandler(c, w, r)
	case http.MethodPut:
		code, err := resourcePostPutHandler(c, w, r)
		if code != http.StatusOK {
			return code, err
		}
		return code, err
	case http.MethodPatch:
		return resourcePatchHandler(c, w, r)
	case http.MethodPost:
		return resourcePostPutHandler(c, w, r)
	}

	return http.StatusNotImplemented, nil
}

func resourceGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// Gets the information of the directory/file.
	f, err := fb.GetInfo(r.URL, c)
	if err != nil {
		return ErrorToHTTP(err, false), err
	}

	// If it's a dir and the path doesn't end with a trailing slash,
	// add a trailing slash to the path.
	if f.IsDir && !strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path = r.URL.Path + "/"
	}

	// If it is a dir, go and serve the listing.
	if f.IsDir {
		c.File = f
		listingHandler(c, w, r)
		return renderJSON(w, f)
	}

	// Tries to get the file type.
	if err = f.SetFileType(true); err != nil {
		return ErrorToHTTP(err, true), err
	}

	// Serve a preview if the file can't be edited or the
	// user has no permission to edit this file. Otherwise,
	// just serve the editor.
	if !f.CanBeEdited() || !c.User.AllowEdit {
		f.Kind = "preview"
		return renderJSON(w, f)
	}

	f.Kind = "editor"

	return renderJSON(w, f)
}

func listingHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	f := c.File
	f.Kind = "listing"

	// Tries to get the listing data.
	if err := f.GetListing(c.User, c.IsRecursive); err != nil {
		return ErrorToHTTP(err, true), err
	}

	listing := f.Listing

	// Defines the cookie scope.
	cookieScope := c.RootURL()
	if cookieScope == "" {
		cookieScope = "/"
	}

	// Copy the query values into the Listing struct
	if sort, order, err := handleSortOrder(w, r, cookieScope); err == nil {
		listing.Sort = sort
		listing.Order = order
	} else {
		return http.StatusBadRequest, err
	}

	listing.ApplySort()
	listing.AllowGeneratePreview = c.Config.AllowGeneratePreview

	return 0, nil
}

func resourceDeleteHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// Prevent the removal of the root directory.
	if r.URL.Path == "/" || !c.User.AllowEdit {
		return http.StatusForbidden, nil
	}
	removePreview(c, r)

	// Remove the file or folder.
	err := c.User.FileSystem.RemoveAll(r.URL.Path)

	if err != nil {
		return ErrorToHTTP(err, true), err
	}

	return http.StatusOK, nil
}
func removePreview(c *fb.Context, r *http.Request) {
	info, err := c.User.FileSystemPreview.Stat(r.URL.Path)
	if err != nil {
		log.Printf("resource: preview file locked or it does not exists %s", err)
		return
	}
	var src string
	if !info.IsDir() {
		src, _ = fileutils.ReplacePrevExt(r.URL.Path)
	} else {
		src = r.URL.Path
	}

	err = c.User.FileSystemPreview.RemoveAll(src)
	if err != nil {
		log.Println(err)
	}
} //rename preview
func modPreview(c *fb.Context, src, dst string, isCopy bool) {
	info, err := c.User.FileSystem.Stat(src)
	if err != nil {
		log.Printf("resource: preview file locked or it does not exists %s", err)
		return
	}
	if !info.IsDir() {
		src, _ = fileutils.ReplacePrevExt(src)
		dst, _ = fileutils.ReplacePrevExt(dst)

	}
	if isCopy {
		c.User.FileSystemPreview.Copy(src, dst)
	} else {
		c.User.FileSystemPreview.Rename(src, dst)
	}
}

func resourcePostPutHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if !c.User.AllowNew && r.Method == http.MethodPost {
		return http.StatusForbidden, nil
	}

	if !c.User.AllowEdit && r.Method == http.MethodPut {
		return http.StatusForbidden, nil
	}

	// Discard any invalid upload before returning to avoid connection
	// reset error.
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
	}()

	// Checks if the current request is for a directory and not a file.
	if strings.HasSuffix(r.URL.Path, "/") {
		// If the method is PUT, we return 405 Method not Allowed, because
		// POST should be used instead.
		if r.Method == http.MethodPut {
			return http.StatusMethodNotAllowed, nil
		}

		// Otherwise we try to create the directory.
		err := c.User.FileSystem.Mkdir(r.URL.Path, 0775)
		c.User.FileSystemPreview.Mkdir(filepath.Join(c.User.Scope, r.URL.Path), 0775)
		return ErrorToHTTP(err, false), err
	}

	// If using POST method, we are trying to create a new file so it is not
	// desirable to override an already existent file. Thus, we check
	// if the file already exists. If so, we just return a 409 Conflict.
	if r.Method == http.MethodPost && r.Header.Get("Action") != "override" {
		if _, err := c.User.FileSystem.Stat(r.URL.Path); err == nil {
			return http.StatusConflict, errors.New("There is already a file on that path")
		}
	}
	// Create/Open the file.
	f, err := c.User.FileSystem.OpenFile(r.URL.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775)
	if err != nil {
		return ErrorToHTTP(err, false), err
	}
	defer f.Close()

	// Copies the new content for the file.
	_, err = io.Copy(f, r.Body)
	if err != nil {
		return ErrorToHTTP(err, false), err
	}

	// Gets the info about the file.
	fi, err := f.Stat()
	if err != nil {
		return ErrorToHTTP(err, false), err
	}
	if !fi.IsDir() {
		inf, err := fb.GetInfo(r.URL, c)
		if err == nil {
			c.File = inf
			modP := fileutils.PreviewPathMod(r.URL.Path, c.User.Scope, c.User.PreviewScope)
			ok, _ := fileutils.Exists(modP)
			if !ok {
				c.GenPreview(modP)
			}
		}

	}
	// Writes the ETag Header.
	etag := fmt.Sprintf(`"%x%x"`, fi.ModTime().UnixNano(), fi.Size())
	w.Header().Set("ETag", etag)

	return http.StatusOK, nil
}

// resourcePatchHandler is the entry point for resource handler.
func resourcePatchHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if !c.User.AllowEdit {
		return http.StatusForbidden, nil
	}

	dst := r.Header.Get("Destination")
	action := r.Header.Get("Action")
	dst, err := url.QueryUnescape(dst)
	if err != nil {
		return ErrorToHTTP(err, true), err
	}

	src := r.URL.Path

	if dst == "/" || src == "/" {
		return http.StatusForbidden, nil
	}

	if action == "copy" {
		modPreview(c, src, dst, true)
		// Copy the file.
		err = c.User.FileSystem.Copy(src, dst)

	} else {
		modPreview(c, src, dst, false)
		// Rename the file.
		err = c.User.FileSystem.Rename(src, dst)

	}

	return ErrorToHTTP(err, true), err
}

// handleSortOrder gets and stores for a Listing the 'sort' and 'order',
// and reads 'limit' if given. The latter is 0 if not given. Sets cookies.
func handleSortOrder(w http.ResponseWriter, r *http.Request, scope string) (sort string, order string, err error) {
	sort = r.URL.Query().Get("sort")
	order = r.URL.Query().Get("order")

	// If the query 'sort' or 'order' is empty, use defaults or any values
	// previously saved in Cookies.
	switch sort {
	case "":
		sort = "name"
		if sortCookie, sortErr := r.Cookie("sort"); sortErr == nil {
			sort = sortCookie.Value
		}
	case "name", "size":
		http.SetCookie(w, &http.Cookie{
			Name:   "sort",
			Value:  sort,
			MaxAge: 31536000,
			Path:   scope,
			Secure: r.TLS != nil,
		})
	}

	switch order {
	case "":
		order = "asc"
		if orderCookie, orderErr := r.Cookie("order"); orderErr == nil {
			order = orderCookie.Value
		}
	case "asc", "desc":
		http.SetCookie(w, &http.Cookie{
			Name:   "order",
			Value:  order,
			MaxAge: 31536000,
			Path:   scope,
			Secure: r.TLS != nil,
		})
	}

	return
}
