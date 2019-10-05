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
		hasSearch := len(c.SearchString) > 0 && len(name) > 0
		var fitUrl bool
		if hasSearch {
			fitUrl = strings.Contains(name, c.SearchString) ||
				strings.Contains(p, c.SearchString)
		}

		var fitType bool
		ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(name))
		hasType := c.Audio || c.Video || c.Pdf || c.Image
		if ok && hasType {
			fitType = t == cnst.IMAGE && c.Image ||
				t == cnst.AUDIO && c.Audio ||
				t == cnst.VIDEO && c.Video ||
				t == cnst.PDF && c.Pdf
		}

		return hasType && fitType && (fitUrl || !hasSearch) || !hasType && fitUrl
	}
	c.IsRecursive = true
	_, r.URL.Path = fb.SplitURL(r.URL.Path)

	return resourceGetHandler(c, w, r, filter)

}
