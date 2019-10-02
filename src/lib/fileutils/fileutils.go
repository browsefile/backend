// Package fileutils implements some useful functions
// to work with the file system.
package fileutils

import (
	"github.com/browsefile/backend/src/cnst"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SlashClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func SlashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

// mimeExt is the sorted list of text mimeExt which
// can be edited.

// getBasedOnExtensions checks if a file can be edited by its mimeExt.
func GetBasedOnExtensions(name string) (res bool, t string) {
	if len(name) == 0 {
		return false, ""
	}
	name = strings.ToLower(name)
	ext := filepath.Ext(name)
	if ext == "" {
		ext = name
	}
	res = strings.EqualFold(".pdf", ext)
	if res {
		return res, cnst.PDF
	}
	for iEx, eArr := range cnst.MIME_EXT {
		for _, e := range eArr {
			res = strings.EqualFold(e, ext)
			if res {
				switch iEx {
				case 0:
					t = cnst.IMAGE
				case 1:
					t = cnst.VIDEO
				case 2:
					t = cnst.AUDIO
				case 3:
					t = cnst.TEXT
				}
				break
			}
		}
		if res {
			break
		}
	}
	if !res {
		//log.Printf("fileutils can't detect type: %s", ext)
		t = cnst.BLOB
	}

	return
}

//should get information about file.
func GetFileInfo(scope, urlPath string) (info os.FileInfo, err error, path string) {
	dir := Dir(scope)
	info, err = dir.Stat(urlPath)
	path = filepath.Join(scope, urlPath)
	if err != nil {
		return info, err, path
	}
	return info, err, path
}
func PreviewPathMod(orig, scope, pScope string) (p string) {
	rPath := strings.TrimPrefix(orig, scope)
	p = filepath.Join(pScope, rPath)
	//replace file extension
	p, _ = ReplacePrevExt(p)
	return
}
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

//modify existing file extension to the preview
func ReplacePrevExt(srcPath string) (path string, t string) {
	extension := filepath.Ext(srcPath)
	if len(extension) > 0 {
		var ext string
		_, t = GetBasedOnExtensions(extension)
		if t == cnst.VIDEO {
			ext = ".gif"
		} else {
			ext = ".jpg"
		}

		path = strings.TrimSuffix(srcPath, extension) + ext
	} else {
		path = srcPath
	}

	return path, t
}

// Will return input and output to be processed to the bash convert/ffmpeg in order to generate preview
func GenPreviewConvertPath(path string, scope string, previewScope string) (outp string, err error) {
	if !strings.EqualFold(filepath.Dir(path), path) {

		outp = filepath.Join(previewScope, strings.TrimPrefix(path, scope))
		outp, _ = ReplacePrevExt(outp)
	}

	return

}
func ModPermission(uid, gid int, path string) (err error) {
	if uid > 0 && gid > 0 {
		err = os.Chown(path, uid, gid)
		if err != nil {
			log.Println(err)
		}
	}
	return err

}

// just in case
func CleanPath(p string) (string, error) {
	p, err := url.QueryUnescape(p)
	if err != nil {
		return "", err
	}

	// Clean the slashes.
	p = SlashClean(p)
	return p, nil
}
