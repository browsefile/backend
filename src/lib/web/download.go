package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
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
		if len(c.FilePaths) == 1 {
			c.URL = c.FilePaths[0]
		}
		c.URL = utils.SlashClean(c.URL)
		if len(c.URL) == 0 {
			return http.StatusBadRequest, cnst.ErrInvalidOption
		}
		c.File, err = c.MakeInfo()
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err
		}

		// If the file isn't a directory, serve it using web.ServeFile. We display it
		// inline if it is requested.
		if !c.File.IsDir {
			return downloadFileHandler(c)
		} else {
			//todo: remove redundant makeInfo for single file
			c.FilePaths = []string{utils.CutUserPath(c.File.Path, c.Config.FilesPath)}
		}
	}
	code, err, infos := prepareFiles(c)
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
func prepareFiles(c *fb.Context) (int, error, []os.FileInfo) {
	var resultFiles = make([]string, 0, len(c.FilePaths))
	var resultInfos = make([]os.FileInfo, 0, len(c.FilePaths))
	var err error
	// If there are files in the query, sanitize their names.
	// Otherwise, just append the current path.
	//todo: allow share to self
	extUsr, _ := c.Config.GetUserByUsername(cnst.GUEST)
	extUsrMod := fb.ToUserModel(extUsr, c.Config)
	for _, p := range c.FilePaths {
		p = utils.SlashClean(p)
		c.URL = p
		if c.IsExternal {
			c.User = extUsrMod
		}
		c.File, err = c.MakeInfo()
		if err != nil {
			return cnst.ErrorToHTTP(err, false), err, nil
		}
		c.IsRecursive = c.File.IsDir
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
func serveDownload(c *fb.Context, infos []os.FileInfo) error {
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
	return utils.ServeArchiveCompress(c.FilePaths, c.Config.FilesPath, c.RESP, infos)
}

//download single file, include preview
func downloadFileHandler(c *fb.Context) (int, error) {
	var err error
	c.File.Path = utils.SlashClean(c.File.Path)

	file, err := os.Open(c.File.Path)
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return http.StatusNotFound, err
	}

	//serve icon
	if len(c.PreviewType) > 0 {
		var prevPath string
		_, prevPath, err = c.ResolveContextUser()
		if c.IsExternal {
			prevPath = filepath.Join(prevPath, c.URL)
		}

		if c.IsShare {
			prevPath, _ = utils.ReplacePrevExt(prevPath)
		} else {
			prevPath = utils.GenPreviewConvertPath(c.File.Path, c.GetUserHomePath(), c.GetUserPreviewPath())
		}

		if !utils.Exists(prevPath) {
			if c.IsShare {
				c.GenSharesPreview(prevPath)
			} else {
				c.GenPreview(prevPath)
			}

		} else {
			c.RESP.Header().Set("Content-Type", utils.GetMimeType(prevPath))
			return servePreview(c, prevPath)
		}
	}
	_, c.File.Type = utils.GetFileType(c.File.Name)
	m := utils.GetMimeType(c.File.Path)
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
