// Package fileutils implements some useful functions
// to work with the file system.
package fileutils

import (
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
var mimeExt = [][]string{{
	".ad", ".ada", ".adoc", ".asciidoc",
	".bas", ".bash", ".bat",
	".c", ".cc", ".cmd", ".conf", ".cpp", ".cr", ".cs", ".css", ".csv",
	".d",
	".f", ".f90",
	".h", ".hh", ".hpp", ".htaccess", ".html",
	".ini",
	".java", ".js", ".json",
	".markdown", ".md", ".mdown", ".mmark",
	".nim",
	".php", ".pl", ".ps1", ".py", ".go",
	".rss", ".rst", ".rtf",
	".sass", ".scss", ".sh", ".sty",
	".tex", ".tml", ".toml", ".txt",
	".vala", ".vapi",
	".xml",
	".yaml", ".yml",
	"caddyfile",
}, {
	".3gp", ".3g2", ".asf", ".wma", ".wmv",
	".avi", ".divx", ".f4v", ".evo", ".flv",
	".MKV", ".MK3D", ".MKA", ".MKS", ".webm",
	".mcf", ".mp4", ".mpg", ".mpeg", ".m2p",
	".ps", ".ts", ".m2ts", ".mxf",
	".mov", ".qt", ".rmvb", ".vob",
}, {
	".jpg", ".png", ".jpeg", ".tiff", ".tif", ".bmp",
	".gif", ".eps", ".raw", ".cr2", ".nef", ".orf", ".sr2",
}, {
	".aa", ".aac", ".mp3", ".aiff", ".amr", ".act", ".aax",
	".au", ".awb", ".flac", ".m4a", ".m4b", ".m4p", ".ra", ".rm", ".wav",
	".alac", ".ogg",
}}

// getBasedOnExtensions checks if a file can be edited by its mimeExt.
func GetBasedOnExtensions(name string) (res bool, t string) {
	name = strings.ToLower(name)
	ext := filepath.Ext(name)
	if ext == "" {
		ext = name
	}
	res = strings.EqualFold(".pdf", ext)
	if res {
		return res, "pdf"
	}
	for iEx, eArr := range mimeExt {
		for _, e := range eArr {
			res = strings.EqualFold(e, ext)
			if res {
				switch iEx {
				case 0:
					t = "text"

				case 1:
					t = "video"

				case 2:
					t = "image"

				case 3:
					t = "audio"

				}
				break
			}
		}
		if res {
			break
		}
	}
	if !res {
		log.Printf("fileutils can't detect type: %s", ext)
		t = "blob"
	}

	return
}

//should get information about original file. Depending on previewType, it will return correct relative path at the file system
func GetFileInfo(scope, urlPath string) (info os.FileInfo, err error, path string, t string) {
	dir := Dir(scope)
	info, err = dir.Stat(urlPath)
	path = filepath.Join(scope, urlPath)
	if err != nil {
		return info, err, path, ""
	}
	return info, err, path, t
}
func PreviewPathMod(orig, scope, pScope string) (p string) {
	dir := Dir(scope)
	rPath := strings.Replace(orig, scope, "", 1)
	info, err := dir.Stat(rPath)
	if err != nil {
		log.Println(err)
	}
	p = filepath.Join(pScope, rPath)
	//replace file extension
	if !info.IsDir() {
		p, _ = ReplacePrevExt(p)
	}
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
	name := filepath.Base(srcPath)
	extension := filepath.Ext(name)
	var ext string
	_, t = GetBasedOnExtensions(extension)
	if t == "video" {
		ext = ".gif"
	} else {
		ext = ".jpg"
	}
	//modify extension and path to the preview
	newName := strings.Replace(name, extension, ext, -1)
	path = strings.Replace(srcPath, name, newName, -1)
	return
}

// Will return input and output to be processed to the bash convert/ffmpeg in order to generate preview
func GenPreviewConvertPath(path string, scope string, previewScope string) (outp string, err error) {
	if !strings.EqualFold(filepath.Dir(path), path) {
		outp = strings.Replace(path, scope, previewScope, 1)
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
