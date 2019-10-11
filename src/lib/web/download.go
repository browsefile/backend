package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// downloadHandler creates an archive in one of the supported formats (zip, tar,
// tar.gz or tar.bz2) and sends it to be downloaded.
func downloadHandler(c *fb.Context) (code int, err error) {
	if len(c.FilePaths) <= 1 {
		p := c.URL
		if len(c.FilePaths) == 1 {
			p, err = fileutils.CleanPath(c.FilePaths[0])
			if err != nil {
				return http.StatusInternalServerError, err
			}
		}
		c.URL = p
		c.File, err = fb.MakeInfo(c)
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err
		}

		// If the file isn't a directory, serve it using web.ServeFile. We display it
		// inline if it is requested.
		if !c.File.IsDir {
			return downloadFileHandler(c)
		} else {
			//todo: remove redundant makeInfo for single file
			c.FilePaths = []string{fileutils.CutUserPath(c.File.Path, c.Config.FilesPath)}
		}
	}
	code, err, infos := prepareFiles(c, c.URL)
	if err != nil {
		log.Println(err)
		return code, err
	}
	err = serveDownload(c, infos)
	if err != nil {
		log.Println(err)
		code = http.StatusNotFound
	}
	return code, err
}

//take c.FilePaths as input, and put absolute path back as a result, also recursively fetch folders for shares/files
func prepareFiles(c *fb.Context, url string) (int, error, []os.FileInfo) {
	var resultFiles = make([]string, 0, len(c.FilePaths))
	var resultInfos = make([]os.FileInfo, 0, len(c.FilePaths))
	// If there are files in the query, sanitize their names.
	// Otherwise, just append the current path.
	for _, p := range c.FilePaths {
		p, err := fileutils.CleanPath(p)
		if err != nil {
			return http.StatusInternalServerError, err, nil
		}
		c.URL = p
		c.File, err = fb.MakeInfo(c)
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err, nil
		}
		c.IsRecursive = true
		infos, paths, err := c.File.GetListing(c)
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err, nil
		}
		for i, inf := range infos {
			if !inf.IsDir() {
				resultFiles = append(resultFiles, paths[i])
				resultInfos = append(resultInfos, infos[i])
			}
		}

	}
	c.FilePaths = resultFiles
	return http.StatusOK, nil, resultInfos
}

//set header as download, also set archive name, after archive
func serveDownload(c *fb.Context, infos []os.FileInfo) (err error) {
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
	c.RESP.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(name))
	return fileutils.ServeArchiveCompress(c.FilePaths, c.Config.FilesPath, c.RESP, infos)
}

//download single file, include preview
func downloadFileHandler(c *fb.Context) (int, error) {
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
		_, prevPath, c.URL, err = fb.ResolvePaths(c)
		if c.IsExternalShare() {
			prevPath = filepath.Join(prevPath, c.URL)
		}

		if c.IsShare {
			prevPath, _ = fileutils.ReplacePrevExt(prevPath)
		} else {
			prevPath = fileutils.PreviewPathMod(c.File.Path, c.GetUserHomePath(), c.GetUserPreviewPath())
		}

		if !fileutils.Exists(prevPath) {
			if c.IsShare && !c.IsExternalShare() {
				c.GenSharesPreview(prevPath)
			} else {
				c.GenPreview(prevPath)
			}

		} else {
			c.RESP.Header().Set("Content-Type", fb.GetMimeType(prevPath))
			return servePreview(c, prevPath)
		}
	}
	c.File.SetFileType(false)
	m := fb.GetMimeType(c.File.Path)
	if len(m) == 0 {
		m = c.File.Type
	}
	c.RESP.Header().Set("Content-Type", m)

	if c.Inline {
		c.RESP.Header().Set("Content-Disposition", "inline")
	} else {
		// As per RFC6266 section 4.3
		c.RESP.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(c.File.Name))
	}
	//serve fullsize file
	if file != nil {
		http.ServeContent(c.RESP, c.REQ, stat.Name(), stat.ModTime(), file)
		return 0, nil
	}

	return http.StatusNotFound, nil
}

func servePreview(c *fb.Context, p string) (int, error) {
	previewFile, err := os.Open(p)
	stat, _ := os.Stat(p)
	defer previewFile.Close()
	if err != nil {
		return http.StatusNotFound, err
	} else {
		http.ServeContent(c.RESP, c.REQ, stat.Name(), stat.ModTime(), previewFile)
		return 0, nil
	}
}
