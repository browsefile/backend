package web

import (
	"errors"
	"fmt"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	resourceMediaFilter = func(c *fb.Context, name, p string) bool {

		var fitType bool
		ok, t := utils.GetFileType(filepath.Ext(name))
		hasType := c.Audio || c.Video || c.Pdf || c.Image
		if ok && hasType {
			fitType = t == cnst.IMAGE && c.Image ||
				t == cnst.AUDIO && c.Audio ||
				t == cnst.VIDEO && c.Video ||
				t == cnst.PDF && c.Pdf

		}
		return hasType && fitType || !hasType
	}
)
// sanitizeURL sanitizes the URL to prevent path transversal
// using utils.SlashClean and adds the trailing slash bar.
func sanitizeURL(url string) string {
	path := utils.SlashClean(url)
	if strings.HasSuffix(url, "/") && path != "/" {
		return path + "/"
	}
	return path
}

func resourceHandler(c *fb.Context) (int, error) {

	c.URL = sanitizeURL(c.URL)

	switch c.Method {
	case http.MethodGet:
		c.FitFilter = func(name, p string) bool {
			return resourceMediaFilter(c, name, p)
		}
		return resourceGetHandler(c)
	case http.MethodDelete:
		return resourceDeleteHandler(c)
	case http.MethodPut:
		code, err := resourcePostPutHandler(c)
		if code != http.StatusOK {
			return code, err
		}
		return code, err
	case http.MethodPatch:
		return resourcePatchHandler(c)
	case http.MethodPost:
		return resourcePostPutHandler(c)
	}

	return http.StatusNotImplemented, nil
}

func resourceGetHandler(c *fb.Context) (int, error) {
	// GetUsers the information of the directory/file.
	f, err := c.MakeInfo()
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}

	// If it is a dir, go and serve the listing.
	if f.IsDir {
		c.File = f
		listingHandler(c)
		return renderJSON(c, f)
	}

	// Tries to get the file type.

	// If the file type is text, save its content.
	_, f.Type = utils.GetFileType(f.Name)

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
		return renderJSON(c, f)
	}

	f.Kind = "editor"

	return renderJSON(c, f)
}

func listingHandler(c *fb.Context) (int, error) {
	c.File.Kind = "listing"

	// Tries to get the listing data.
	if err := c.File.ProcessList(c); err != nil {
		return cnst.ErrorToHTTP(err, true), err
	}
	// Copy the query values into the Listing struct
	if err := HandleSortOrder(c, "/"); err == nil {
		c.File.Listing.Sort = c.Sort
		c.File.Listing.Order = c.Order
	} else {
		return http.StatusBadRequest, err
	}

	c.File.Listing.ApplySort()
	c.File.Listing.AllowGeneratePreview = len(c.Config.ScriptPath) > 0

	return 0, nil
}

func resourceDeleteHandler(c *fb.Context) (int, error) {
	// Prevent the removal of the root directory.
	if c.URL == "/" || !c.User.AllowEdit {
		return http.StatusForbidden, nil
	}
	removePreview(c)

	// Remove the file or folder.
	err := c.User.FileSystem.RemoveAll(c.URL)

	if err != nil {
		return cnst.ErrorToHTTP(err, true), err
	}
	//delete share
	for _, itm := range findShare(c.User.UserConfig, c.URL) {

		if c.User.DeleteShare(itm.Path) {
			_ = c.Config.Update(c.User.UserConfig)
		}

	}

	return http.StatusOK, nil
}
func findShare(u *config.UserConfig, p string) (res []*config.ShareItem) {
	for _, itm := range u.GetShares(p, true) {
		itmPath := strings.TrimSuffix(itm.Path, "/")
		itmPath = strings.TrimPrefix(itmPath, "/")
		delPath := strings.TrimSuffix(p, "/")
		delPath = strings.TrimPrefix(delPath, "/")
		//check if it is not sub path from share
		if len(itmPath) == len(delPath) || strings.HasPrefix(itmPath, delPath) {
			res = append(res, itm)
		}
	}
	return res
}
func removePreview(c *fb.Context) {
	info, err := c.User.FileSystemPreview.Stat(c.URL)
	if err != nil {
		//log.Printf("resource: preview file locked or it does not exists %s", err)
		return
	}
	var src string
	if !info.IsDir() {
		src, _ = utils.ReplacePrevExt(c.URL)
	} else {
		src = c.URL
	}

	err = c.User.FileSystemPreview.RemoveAll(src)
	if err != nil {
		log.Println(err)
	}
} //rename preview
func modPreview(c *fb.Context, src, dst string, isCopy bool) {
	info, err := c.User.FileSystem.Stat(src)
	_, t := utils.GetFileType(src)
	if err != nil {
		//log.Printf("resource: preview file locked or it does not exists %s", err)
		return
	}
	if t == cnst.IMAGE || t == cnst.VIDEO {
		if !info.IsDir() {
			src, _ = utils.ReplacePrevExt(src)
			dst, _ = utils.ReplacePrevExt(dst)
		}
		if isCopy {
			c.User.FileSystemPreview.Copy(src, dst, c.User.UID, c.User.GID)
		} else {
			c.User.FileSystemPreview.Rename(src, dst)
		}
	}
}

func resourcePostPutHandler(c *fb.Context) (int, error) {
	if !c.User.AllowNew && c.Method == http.MethodPost {
		return http.StatusForbidden, nil
	}

	if !c.User.AllowEdit && c.Method == http.MethodPut {
		return http.StatusForbidden, nil
	}

	// Discard any invalid upload before returning to avoid connection
	// reset error.
	defer func() {
		io.Copy(ioutil.Discard, c.REQ.Body)
	}()

	// Checks if the current request is for a directory and not a file.
	if strings.HasSuffix(c.URL, "/") {
		// If the method is PUT, we return 405 Method not Allowed, because
		// POST should be used instead.
		if c.Method == http.MethodPut {
			return http.StatusMethodNotAllowed, nil
		}

		// Otherwise we try to create the directory.
		err := c.User.FileSystem.Mkdir(c.URL, cnst.PERM_DEFAULT, c.User.UID, c.User.GID)
		if err != nil {
			p := filepath.Join(c.GetUserHomePath(), c.URL)
			err = os.Chown(p, c.User.UID, c.User.GID)
			if err != nil {
				c.User.FileSystemPreview.Mkdir(p, cnst.PERM_DEFAULT, c.User.UID, c.User.GID)
			}
			if !os.IsPermission(err) {
				log.Println(err)
			}
		}
		return cnst.ErrorToHTTP(err, false), err
	}

	// If using POST method, we are trying to create a new file so it is not
	// desirable to override an already existent file. Thus, we check
	// if the file already exists. If so, we just return a 409 Conflict.
	if c.Method == http.MethodPost && !c.Override {
		if _, err := c.User.FileSystem.Stat(c.URL); err == nil {
			return http.StatusConflict, errors.New("There is already a file on that path")
		}
	}
	// Create/Open the file.
	f, err := c.User.FileSystem.OpenFile(c.URL, os.O_RDWR|os.O_CREATE|os.O_TRUNC, cnst.PERM_DEFAULT, c.User.UID, c.User.GID)
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}
	defer f.Close()

	// Copies the new content for the file.
	_, err = io.Copy(f, c.REQ.Body)
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}

	// GetUsers the info about the file.
	fi, err := f.Stat()
	if err != nil {
		return cnst.ErrorToHTTP(err, false), err
	}
	if !fi.IsDir() {
		inf, err := c.MakeInfo()
		if err == nil {
			c.File = inf
			modP := utils.GenPreviewConvertPath(c.URL, c.GetUserHomePath(), c.GetUserPreviewPath())
			if !utils.Exists(modP) {
				c.GenPreview(modP)
			}
		}

	}
	// Writes the ETag Header.
	etag := fmt.Sprintf(`"%x%x"`, fi.ModTime().UnixNano(), fi.Size())
	c.RESP.Header().Set("ETag", etag)

	return http.StatusOK, nil
}

// resourcePatchHandler is the entry point for resource handler.
func resourcePatchHandler(c *fb.Context) (int, error) {
	if !c.User.AllowEdit {
		return http.StatusForbidden, nil
	}
	dst, err := url.QueryUnescape(c.Destination)
	if err != nil {
		return cnst.ErrorToHTTP(err, true), err
	}
	action := c.Action
	src := c.URL

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
			for _, itm := range findShare(c.User.UserConfig, c.URL) {
				if c.User.DeleteShare(itm.Path) {
					_ = c.Config.Update(c.User.UserConfig)
				}
			}
		}

	}

	return cnst.ErrorToHTTP(err, true), err
}

// HandleSortOrder gets and stores for a Listing the 'sort' and 'order',
// and reads 'limit' if given. The latter is 0 if not given. Sets cookies.
func HandleSortOrder(c *fb.Context, scope string) (err error) {

	// If the query 'sort' or 'order' is empty, use defaults or any values
	// previously saved in Cookies.
	switch c.Sort {
	case "":
		c.Sort = "name"
		if sortCookie, sortErr := c.REQ.Cookie("sort"); sortErr == nil {
			c.Sort = sortCookie.Value
		}
	case "name", "size":
		http.SetCookie(c.RESP, &http.Cookie{
			Name:   "sort",
			Value:  c.Sort,
			MaxAge: 31536000,
			Path:   scope,
			Secure: c.REQ.TLS != nil,
		})
	}

	switch c.Order {
	case "":
		c.Order = "asc"
		if orderCookie, orderErr := c.REQ.Cookie("order"); orderErr == nil {
			c.Order = orderCookie.Value
		}
	case "asc", "desc":
		http.SetCookie(c.RESP, &http.Cookie{
			Name:   "order",
			Value:  c.Order,
			MaxAge: 31536000,
			Path:   scope,
			Secure: c.REQ.TLS != nil,
		})
	}

	return
}
