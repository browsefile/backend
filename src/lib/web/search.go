package web

import (
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"net/http"
	"path/filepath"
	"strings"
)

func searchHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	filter := func(name, p string) bool {
		fitUrl := strings.Contains(strings.ToLower(p), strings.ToLower(c.SearchString))

		var fitType bool
		ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(name))
		hasType := c.Audio || c.Video || c.Pdf || c.Image
		if ok && hasType {
			fitType = t == "image" && c.Image ||
				t == "audio" && c.Audio ||
				t == "video" && c.Video ||
				t == "pdf"

		}
		return hasType && fitType && fitUrl || !hasType && fitUrl
	}
	c.IsRecursive = true
	isShares := len(c.RootHash) > 0 || len(c.ShareType) > 0

	_, r.URL.Path = splitURL(r.URL.Path)

	if isShares {
		return shareGetHandler(c, w, r, filter)
	} else {
		return resourceGetHandler(c, w, r, filter)
	}

}
