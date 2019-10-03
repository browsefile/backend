package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

// downloadHandler creates an archive in one of the supported formats (zip, tar,
// tar.gz or tar.bz2) and sends it to be downloaded.
func downloadHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var err error
	if len(c.FilePaths) <= 1 {
		p := r.URL.Path
		if len(c.FilePaths) == 1 {
			p, err = fileutils.CleanPath(c.FilePaths[0])
			if err != nil {
				return http.StatusInternalServerError, err
			}
		}
		c.File, err = fb.MakeInfo(p, r.URL.String(), c)
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err
		}

		// If the file isn't a directory, serve it using web.ServeFile. We display it
		// inline if it is requested.
		if !c.File.IsDir {
			return downloadFileHandler(c, w, r)
		} else {
			return 0, serveDownload(c, w, []string{c.File.Path})
		}
	} else {
		var files = make([]string, 0, len(c.FilePaths))
		// If there are files in the query, sanitize their names.
		// Otherwise, just append the current path.
		for _, name := range c.FilePaths {

			// Unescape the name.
			if c.IsExternalShare() {

			} else {
				name, err = fileutils.CleanPath(name)
			}

			if err != nil {
				return http.StatusInternalServerError, err
			}

			c.File, err = fb.MakeInfo(name, r.URL.String(), c)
			if err != nil {
				return cnst.ErrorToHTTP(err, false), err
			}
			files = append(files, c.File.Path)
		}

		return 0, serveDownload(c, w, files)
	}
	return 0, nil
}
func serveDownload(c *fb.Context, w http.ResponseWriter, files []string) (err error) {
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
	return cmd.Run()

}

func downloadFileHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var err error
	if c.File.Path, err = fileutils.CleanPath(c.File.Path); err != nil {
		if err != nil {
			return http.StatusNotFound, err
		}
	}

	file, err := os.Open(c.File.Path)
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return http.StatusNotFound, err
	}

	//serve icon
	if len(c.PreviewType) > 0 {
		var prevPath string
		_, prevPath, r.URL.Path, err = fb.ResolvePaths(c, r.URL.Path)
		if c.IsExternalShare() {
			prevPath = filepath.Join(prevPath, r.URL.Path)
		}

		if c.IsShare {
			prevPath, _ = fileutils.ReplacePrevExt(prevPath)
		} else {
			prevPath = fileutils.PreviewPathMod(c.File.Path, c.GetUserHomePath(), c.GetUserPreviewPath())
		}

		ok, _ := fileutils.Exists(prevPath)
		if !ok {
			if c.IsShare && !c.IsExternalShare() {
				c.GenSharesPreview(prevPath)
			} else {
				c.GenPreview(prevPath)
			}

		} else {
			w.Header().Set("Content-Type", getMimeType(prevPath))
			return servePreview(c, w, r, prevPath)
		}
	}
	c.File.SetFileType(false)
	m := getMimeType(c.File.Path)
	if len(m) == 0 {
		m = c.File.Type
	}
	w.Header().Set("Content-Type", m)

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
func getMimeType(f string) string {
	m := mime.TypeByExtension(filepath.Ext(f))
	return m
}
func servePreview(c *fb.Context, w http.ResponseWriter, r *http.Request, p string) (int, error) {
	previewFile, err := os.Open(p)
	stat, _ := os.Stat(p)
	defer previewFile.Close()
	if err != nil {
		return http.StatusNotFound, err
	} else {
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), previewFile)
		return 0, nil
	}
}
