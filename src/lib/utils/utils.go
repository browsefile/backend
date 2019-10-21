package utils

import (
	"github.com/browsefile/backend/src/cnst"
	"mime"
	"path/filepath"
	"strings"
)

// SplitURL splits the path and returns everything that stands
// before the first slash and everything that goes after.
func SplitURL(path string) (int, string) {
	if path == "" {
		return 0, ""
	}

	path = strings.TrimPrefix(path, "/")

	i := strings.Index(path, "/")
	if i == -1 {
		return 0, path
	}

	return parseRouter(path[0:i]), path[i:]
}

func parseRouter(r string) (res int) {
	switch r {
	case "download":
		res = cnst.R_DOWNLOAD
	case "resource":
		res = cnst.R_RESOURCE
	case "users":
		res = cnst.R_USERS
	case "settings":
		res = cnst.R_SETTINGS
	case "shares":
		res = cnst.R_SHARES
	case "search":
		res = cnst.R_SEARCH
	case "playlist":
		res = cnst.R_PLAYLIST

	default:
		res = 0
	}
	return
}
func GetMimeType(f string) string {
	m := mime.TypeByExtension(filepath.Ext(f))
	return m
}
