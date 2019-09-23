package lib

import (
	"strings"
)

func FixNonStandardURIEnc(p string) (rs string) {
	//yeah, some browsers unescape, someone escape, whatever
	/*if strings.Contains(p, "%") {
		var err error
		rs, err = url.QueryUnescape(p)
		if err != nil {
			log.Println(err)
		}

	} else if strings.Contains(p, " ") {
		rs = url.QueryEscape(p)
	} else {
		rs = p
	}*/
	return p
}

// SplitURL splits the path and returns everything that stands
// before the first slash and everything that goes after.
func SplitURL(path string) (string, string) {
	if path == "" {
		return "", ""
	}

	path = strings.TrimPrefix(path, "/")

	i := strings.Index(path, "/")
	if i == -1 {
		return "", path
	}

	return path[0:i], path[i:]
}

