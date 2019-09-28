package web

import (
	"errors"
	"fmt"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
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
		return resourceGetHandler(c, w, r, func(name, p string) bool {

			var fitType bool
			ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(name))
			hasType := c.Audio || c.Video || c.Pdf || c.Image
			if ok && hasType {
				fitType = t == cnst.IMAGE && c.Image ||
					t == cnst.AUDIO && c.Audio ||
					t == cnst.VIDEO && c.Video ||
					t == cnst.PDF

			}
			return hasType && fitType || !hasType
		})
	case http.MethodDelete:
		return resourceDeleteHandler(c, w, r)
	case http.MethodPut:
		code, err := resourcePostPutHandler(c, w, r)
		if code != http.StatusOK {
			return code, err
		}
		return code, err
	case http.MethodPatch:
		return resourcePatchHandler(c, r)
	case http.MethodPost:
		return resourcePostPutHandler(c, w, r)
	}

	return http.StatusNotImplemented, nil
}

func resourceGetHandler(c *fb.Context, w http.ResponseWriter, r *http.Request, fitFilter fb.FitFilter) (int, error) {
	// GetUsers the information of the directory/file.
	f, err := fb.MakeInfo(r.URL, c)
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}

	// If it's a dir and the path doesn't end with a trailing slash,
	// add a trailing slash to the path.
	if f.IsDir && !strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path = r.URL.Path + "/"
	}

	// If it is a dir, go and serve the listing.
	if f.IsDir {
		c.File = f
		listingHandler(c, w, r, fitFilter)
		return renderJSON(w, f)
	}

	// Tries to get the file type.

	// If the file type is text, save its content.
	f.SetFileType(true)

	if f.Type == cnst.TEXT {
		var content []byte
		//todo: fix me, what if file too big ?
		content, err = ioutil.ReadFile(f.Path)
		if err != nil {
			return cnst.ErrorToHTTP(err, true), err
		}

		f.Content = string(content)
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

func listingHandler(c *fb.Context, w http.ResponseWriter, r *http.Request, fitFilter fb.FitFilter) (int, error) {
	c.File.Kind = "listing"

	// Tries to get the listing data.
	if err := c.File.GetListing(c, fitFilter); err != nil {
		return cnst.ErrorToHTTP(err, true), err
	}
	// Copy the query values into the Listing struct
	if err := HandleSortOrder(c, w, r, "/"); err == nil {
		c.File.Listing.Sort = c.Sort
		c.File.Listing.Order = c.Order
	} else {
		return http.StatusBadRequest, err
	}

	c.File.Listing.ApplySort()
	c.File.Listing.AllowGeneratePreview = len(c.Config.ScriptPath) > 0

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
		return cnst.ErrorToHTTP(err, true), err
	}
	//delete share
	if itm := findShare(c.User.UserConfig, r.URL.Path); itm != nil {
		c.Config.DeleteShare(c.User.UserConfig, itm.Path)
	}

	return http.StatusOK, nil
}
func findShare(u *config.UserConfig, p string) *config.ShareItem {
	//delete share
	itm := u.GetShare(p, true)
	if itm != nil {
		itmPath := strings.TrimSuffix(itm.Path, "/")
		itmPath = strings.TrimPrefix(itmPath, "/")
		delPath := strings.TrimSuffix(p, "/")
		delPath = strings.TrimPrefix(delPath, "/")
		//check if it is not sub path from share
		if len(itmPath) == len(delPath) || strings.HasPrefix(itmPath, delPath) {
			return itm
		}
	}
	return nil
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
	_, t := fileutils.GetBasedOnExtensions(src)
	if err != nil {
		log.Printf("resource: preview file locked or it does not exists %s", err)
		return
	}
	if t == cnst.IMAGE || t == cnst.VIDEO {
		if !info.IsDir() {
			src, _ = fileutils.ReplacePrevExt(src)
			dst, _ = fileutils.ReplacePrevExt(dst)
		}
		if isCopy {
			c.User.FileSystemPreview.Copy(src, dst, c.User.UID, c.User.GID)
		} else {
			c.User.FileSystemPreview.Rename(src, dst)
		}
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
		err := c.User.FileSystem.Mkdir(r.URL.Path, 0775, c.User.UID, c.User.GID)

		p := filepath.Join(c.GetUserHomePath(), r.URL.Path)
		os.Chown(p, c.User.UID, c.User.GID)
		c.User.FileSystemPreview.Mkdir(p, 0775, c.User.UID, c.User.GID)
		return cnst.ErrorToHTTP(err, false), err
	}

	// If using POST method, we are trying to create a new file so it is not
	// desirable to override an already existent file. Thus, we check
	// if the file already exists. If so, we just return a 409 Conflict.
	if r.Method == http.MethodPost && !c.Override {
		if _, err := c.User.FileSystem.Stat(r.URL.Path); err == nil {
			return http.StatusConflict, errors.New("There is already a file on that path")
		}
	}
	// Create/Open the file.
	f, err := c.User.FileSystem.OpenFile(r.URL.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775, c.User.UID, c.User.GID)
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}
	defer f.Close()

	// Copies the new content for the file.
	_, err = io.Copy(f, r.Body)
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}

	// GetUsers the info about the file.
	fi, err := f.Stat()
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}
	if !fi.IsDir() {
		inf, err := fb.MakeInfo(r.URL, c)
		if err == nil {
			c.File = inf
			modP := fileutils.PreviewPathMod(r.URL.Path, c.GetUserHomePath(), c.GetUserPreviewPath())
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
func resourcePatchHandler(c *fb.Context, r *http.Request) (int, error) {
	if !c.User.AllowEdit {
		return http.StatusForbidden, nil
	}
	dst, err := url.QueryUnescape(c.Destination)
	if err != nil {
		return cnst.ErrorToHTTP(err, true), err
	}
	action := c.Action
	src := r.URL.Path

	if dst == "/" || src == "/" {
		return http.StatusForbidden, nil
	}

	if action == "copy" {
		modPreview(c, src, dst, true)
		// Copy the file.
		err = c.User.FileSystem.Copy(src, dst, c.User.UID, c.User.GID)

	} else {
		modPreview(c, src, dst, false)
		// Rename the file.
		err = c.User.FileSystem.Rename(src, dst)
		if err == nil {
			//check if share exists
			if itm := findShare(c.User.UserConfig, r.URL.Path); itm != nil {
				//just delete share, since it is not actual anymore
				c.Config.DeleteShare(c.User.UserConfig, itm.Path)
			}
		}

	}

	return cnst.ErrorToHTTP(err, true), err
}

// HandleSortOrder gets and stores for a Listing the 'sort' and 'order',
// and reads 'limit' if given. The latter is 0 if not given. Sets cookies.
func HandleSortOrder(c *fb.Context, w http.ResponseWriter, r *http.Request, scope string) (err error) {

	// If the query 'sort' or 'order' is empty, use defaults or any values
	// previously saved in Cookies.
	switch c.Sort {
	case "":
		c.Sort = "name"
		if sortCookie, sortErr := r.Cookie("sort"); sortErr == nil {
			c.Sort = sortCookie.Value
		}
	case "name", "size":
		http.SetCookie(w, &http.Cookie{
			Name:   "sort",
			Value:  c.Sort,
			MaxAge: 31536000,
			Path:   scope,
			Secure: r.TLS != nil,
		})
	}

	switch c.Order {
	case "":
		c.Order = "asc"
		if orderCookie, orderErr := r.Cookie("order"); orderErr == nil {
			c.Order = orderCookie.Value
		}
	case "asc", "desc":
		http.SetCookie(w, &http.Cookie{
			Name:   "order",
			Value:  c.Order,
			MaxAge: 31536000,
			Path:   scope,
			Secure: r.TLS != nil,
		})
	}

	return
}
