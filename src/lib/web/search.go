package web

import (
	"github.com/browsefile/backend/src/cnst"
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
			fitType = t == cnst.IMAGE && c.Image ||
				t == cnst.AUDIO && c.Audio ||
				t == cnst.VIDEO && c.Video ||
				t == cnst.PDF

		}
		return hasType && fitType && fitUrl || !hasType && fitUrl
	}
	c.IsRecursive = true
	isShares := len(c.RootHash) > 0 || len(c.ShareType) > 0

	_, r.URL.Path = fb.SplitURL(r.URL.Path)

	if isShares {
		return shareGetHandler(c, w, r, filter)
	} else {
		return resourceGetHandler(c, w, r, filter)
	}

}
