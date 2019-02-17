package web

import (
	"github.com/browsefile/backend/src/config"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// downloadHandler creates an archive in one of the supported formats (zip, tar,
// tar.gz or tar.bz2) and sends it to be downloaded.
func downloadHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// If the file isn't a directory, serve it using web.ServeFile. We display it
	// inline if it is requested.
	if !c.File.IsDir {
		return downloadFileHandler(c, w, r)
	}
	files := []string{"-rqj", "-"}
	names := strings.Split(r.URL.Query().Get("files"), ",")

	// If there are files in the query, sanitize their names.
	// Otherwise, just append the current path.
	if len(names) != 0 {
		for _, name := range names {
			// Unescape the name.
			name, err := url.QueryUnescape(name)
			if err != nil {
				return http.StatusInternalServerError, err
			}

			// Clean the slashes.
			name = fileutils.SlashClean(name)
			files = append(files, filepath.Join(c.File.Path, name))
		}
	} else {
		files = append(files, c.File.Path)
	}
	return 0, serveDownload(c, w, files)
}
func downloadSharesHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var paths []string

	queryValues := r.URL.Query()
	fParm := queryValues.Get("files")
	if len(fParm) > 0 {
		fArr := strings.Split(fParm, ",")
		for _, fp := range fArr {
			urlPath, err := url.Parse(fp)
			if err != nil {
				return http.StatusNotFound, nil
			}
			uname := urlPath.Query().Get("share")
			itm, usr := config.GetShare(c.User.Username, uname, urlPath.Path)
			//share found and allowed
			if itm != nil {
				paths = append(paths, fileutils.SlashClean(filepath.Join(usr.Scope, urlPath.Path)))
			}
		}

	} else {
		var err error

		item, uc := config.GetShare(c.User.Username, c.ShareUser, r.URL.Path)

		if item != nil && len(item.Path) > 0 {
			c.User = &fb.UserModel{uc, uc.Username, fileutils.Dir(uc.Scope), fileutils.Dir(uc.PreviewScope)}

		} else if err != nil {
			return http.StatusNotFound, err
		}
		c.File, err = fb.GetInfo(r.URL, c)
		if err != nil {
			return http.StatusNotFound, err
		}

		return downloadFileHandler(c, w, r)
	}

	files := []string{"-rqj", "-"}

	// If there are files in the query, sanitize their names.
	// Otherwise, just append the current path.
	if len(paths) != 0 {
		for _, name := range paths {
			// Unescape the name.
			files = append(files, name)
		}
	} else {
		files = append(files, c.File.Path)
	}

	return 0, serveDownload(c, w, files)
}

func serveDownload(c *fb.Context, w http.ResponseWriter, files []string) error {
	// Defines the file name.
	name := ""
	if c.File != nil {
		name = c.File.Name
	}
	if name == "." || name == "" {
		name = "archive.zip"
	} else {
		name += ".zip"
	}
	w.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(name))
	pr, pw := io.Pipe()
	cmd := exec.Command("zip", files...)
	cmd.Stdout = pw
	cmd.Stderr = os.Stderr
	go func() {
		defer pr.Close()
		// copy the data written to the PipeReader via the cmd to stdout
		if _, err := io.Copy(w, pr); err != nil {
			log.Println(err)
		}
	}()
	err := cmd.Run()
	return err
}

func downloadFileHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	file, err := os.Open(c.File.Path)
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return http.StatusNotFound, err
	}
	if len(c.PreviewType) > 0 {
		modP := fileutils.PreviewPathMod(c.File.Path, c.User.Scope, c.User.PreviewScope)
		ok, err := fileutils.Exists(modP)
		if !ok {
			c.GenPreview(modP)
			file, err = os.Open(c.File.Path)
			if err != nil {
				return http.StatusNotFound, err
			}
		}
		file, err = os.Open(modP)
	}

	if r.URL.Query().Get("inline") == "true" {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		// As per RFC6266 section 4.3
		w.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(c.File.Name))
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)

	return 0, nil
}
