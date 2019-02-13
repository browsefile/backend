package web

import (
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
	// Defines the file name.
	name := c.File.Name
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

	return 0, err
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
