// Package utils implements some useful functions
// to work with the file system.
package utils

import (
	"archive/zip"
	"github.com/browsefile/backend/src/cnst"
	"io"
	"log"
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
func GetFileType(name string) (res bool, t string) {
	if len(name) == 0 {
		return false, ""
	}
	name = strings.ToLower(name)
	ext := filepath.Ext(name)
	if ext == "" {
		ext = name
	}
	res = ".pdf" == ext
	if res {
		return res, cnst.PDF
	}
	for iEx, eArr := range cnst.MIME_EXT {
		for _, e := range eArr {
			res = e == ext
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
		//log.Printf("utils can't detect type: %s", ext)
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
		return nil, err, ""
	}
	return info, err, path
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

//modify existing file extension to the preview
func ReplacePrevExt(srcPath string) (path string, t string) {
	extension := filepath.Ext(srcPath)
	if len(extension) > 0 {
		var ext string
		_, t = GetFileType(extension)
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
func GenPreviewConvertPath(path string, scope string, previewScope string) (outp string) {
	if filepath.Dir(path) != path {

		outp = filepath.Join(previewScope, strings.TrimPrefix(path, scope))
		outp, _ = ReplacePrevExt(outp)
	}

	return

}
func ModPermission(uid, gid int, path string) (err error) {
	if uid > 0 && gid > 0 {
		err = os.Chown(path, uid, gid)
	}
	return err

}

//write archive file to writer, paths - absolute files paths, filesFolder - absolute path for users folder, this method will trim user folder path from archive
func ServeArchiveCompress(paths []string, filesFolder string, writer io.Writer, infos []os.FileInfo) (err error) {
	archive := zip.NewWriter(writer)
	defer func() {
		err = archive.Flush()
		if err != nil {
			log.Println(err)
		}
		err = archive.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	for i, f := range paths {
		p := CutUserPath(f, filesFolder)
		file, err := os.OpenFile(f, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		//ignore compression for now
		header := &zip.FileHeader{Name: p, Method: zip.Store, Modified: infos[i].ModTime()}
		if err != nil {
			return err
		}
		fWrt, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(fWrt, file)
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}

	}
	return err
}
func ResolveSymlink(p string) (inf os.FileInfo, realPath string, err error) {
	realPath, err = filepath.EvalSymlinks(p)
	if err != nil {
		return nil, "", err
	}
	inf, err = os.Stat(realPath)
	return
}

/**
removes users path, and trim next prefix userName/files
filesPath - path for users data directory
*/
func CutUserPath(p, filesPath string) string {
	p = strings.TrimPrefix(p, filesPath)
	arr := strings.SplitN(p, "/", 4)
	return arr[len(arr)-1]
}
