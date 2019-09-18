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
)

// downloadHandler creates an archive in one of the supported formats (zip, tar,
// tar.gz or tar.bz2) and sends it to be downloaded.
func downloadHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// If the file isn't a directory, serve it using web.ServeFile. We display it
	// inline if it is requested.
	if !c.File.IsDir {
		return downloadFileHandler(c, w, r)
	}
	files := []string{}
	names := c.FilePaths

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
	if len(c.FilePaths) > 0 {
		fArr := c.FilePaths
		origUsr := c.ShareType
		for _, fp := range fArr {
			urlPath, err := url.Parse(fp)
			if err != nil {
				return http.StatusNotFound, nil
			}
			q := urlPath.Query()
			c.ShareType = q.Get("share")
			c.RootHash = q.Get("rootHash")

			itm, usr := getShare(urlPath.Path, c)
			//share found and allowed
			if itm != nil {
				var p string
				if c.User.IsGuest() || len(c.RootHash) > 0 {
					p = filepath.Join(c.Config.GetUserHomePath(usr.Username), itm.Path, urlPath.Path)
				} else {
					p = filepath.Join(c.Config.GetUserHomePath(usr.Username), urlPath.Path)
				}
				paths = append(paths, fileutils.SlashClean(p))
			}
		}
		c.ShareType = origUsr
		//serve only 1 file without zip

		if len(paths) == 1 {
			//todo: fix file creation
			info, err, _, _ := fileutils.GetFileInfo(paths[0], "")
			if err == nil && !info.IsDir() {
				c.File = &fb.File{
					Path: paths[0],
					Name: filepath.Base(paths[0]),
				}
				return downloadFileHandler(c, w, r)
			}
		}

	} else {
		var err error
		item, uc := getShare(r.URL.Path, c)

		if item != nil && len(item.Path) > 0 {
			c.User = fb.ToUserModel(uc, c.Config)

		} else if err != nil {
			return http.StatusNotFound, err
		}
		if c.IsExternalShare() {
			r.URL.Path = filepath.Join(item.Path, r.URL.Path)
		}
		c.File, err = fb.MakeInfo(r.URL, c)
		if err != nil {
			return http.StatusNotFound, err
		}

		return downloadFileHandler(c, w, r)
	}

	files := []string{}

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
func getShare(p string, c *fb.Context) (*config.ShareItem, *config.UserConfig) {
	//no direct way for usual share get, to guest users
	if c.User.IsGuest() || len(c.RootHash) > 0 {
		return c.Config.GetExternal(c.RootHash)
	} else {
		return config.GetShare(c.User.Username, c.ShareType, p)
	}
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

	cmd := exec.Command("zip", append([]string{"-0rqj", "-"}, files...)...)
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
	//serve icon
	if len(c.PreviewType) > 0 {
		modP := fileutils.PreviewPathMod(c.File.Path, c.GetUserHomePath(), c.GetUserPreviewPath())
		ok, _ := fileutils.Exists(modP)
		if !ok {
			c.GenPreview(modP)
		}
		previewFile, err := os.Open(modP)
		defer previewFile.Close()
		if err != nil {
			return http.StatusNotFound, err
		} else {
			http.ServeContent(w, r, stat.Name(), stat.ModTime(), previewFile)
			return 0, nil
		}

	}

	if c.Inline {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		// As per RFC6266 section 4.3
		w.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(c.File.Name))
	}
	//serve fullsize file
	if file != nil {
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
		return 0, nil
	}

	return http.StatusNotFound, nil
}
