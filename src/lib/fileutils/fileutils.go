// Package fileutils implements some useful functions
// to work with the file system.
package fileutils

import (
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
	".php", ".pl", ".ps1", ".py",
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
	".ps", ".ts", ".m2ts", ".mxf", ".ogg",
	".mov", ".qt", ".rmvb", ".vob",
}, {
	".jpg", ".png", ".jpeg", ".tiff", ".tif", ".bmp",
	".gif", ".eps", ".raw", ".cr2", ".nef", ".orf", ".sr2",
}, {
	".aa", ".aac", ".mp3", ".aiff", ".amr", ".act", ".aax",
	".au", ".awb", ".flac", ".m4a", ".m4b", ".m4p", ".ra", ".rm", ".wav",
	".alac",
}}

// getBasedOnExtensions checks if a file can be edited by its mimeExt.
func GetBasedOnExtensions(name string) (res bool, t string) {
	ext := filepath.Ext(name)
	if ext == "" {
		ext = name
	}
	name = strings.ToLower(name)
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

	return
}

//should get information about original file. Depending on previewType, it will return correct relative path at the file system
func GetFileInfo(scope, pScope string, urlPath string, previewType string) (info os.FileInfo, err error, path string, t string) {
	dir := Dir(scope)
	info, err = dir.Stat(urlPath)
	path = filepath.Join(scope, urlPath)
	if err != nil {
		return info, err, path, ""
	}

	if len(previewType) > 0 {
		//replace file extension
		if !info.IsDir() {
			path, t = ReplacePrevExt(scope, urlPath)
		}
		path = filepath.Join(pScope, path)

	}
	return info, err, path, t
}
//modify existing file extension to the preview
func ReplacePrevExt(scope string, srcPath string) (path string, t string) {
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
	path = filepath.Join(scope, strings.Replace(srcPath, name, newName, -1))
	return
}

/*func GetBasedOnContent(path string) (content []byte, mimetype string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, "", err
	}

	// Tries to get the file mimetype using its first
	// 512 bytes.
	mimetype = web.DetectContentType(buffer[:n])

	if strings.HasPrefix(mimetype, "video") {
		mimetype = "video"
	}
	if strings.HasPrefix(mimetype, "audio") {
		mimetype = "audio"
	}

	if strings.HasPrefix(mimetype, "image") {
		mimetype = "image"
	}

	if strings.HasPrefix(mimetype, "text") || strings.HasPrefix(mimetype, "application/javascript") {
		mimetype = "text"
	} else {
		// If the type isn't text (and is blob for example), it will check some
		// common types that are mistaken not to be text.
		mimetype = "blob"
	}
	return buffer, mimetype, err

}*/

// Will return input and output to be processed to the bash convert/ffmpeg in order to generate preview
func GenPreviewConvertPath(path string, scope string, previewScope string) (inp, outp string, err error) {
	var info os.FileInfo

	info, err = os.Stat(filepath.Join(scope, path))
	if err != nil {
		return "", "", err
	}
	inp = filepath.Join(scope, path)

	if !info.Mode().IsDir() {
		isOk, t := GetBasedOnExtensions(info.Name())

		if isOk && t != "text" || t == "" {
			outp, _ = ReplacePrevExt(scope, path)
			outp = filepath.Join(previewScope, outp)
		}

	}

	return

}
