package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"path/filepath"
	"strings"
)

func searchHandler(c *fb.Context) (int, error) {
	c.FitFilter = func(name, p string) bool {
		hasSearch := len(c.SearchString) > 0 && len(name) > 0
		var fitUrl bool
		if hasSearch {
			fitUrl = strings.Contains(name, c.SearchString) ||
				strings.Contains(p, c.SearchString)
		}

		var fitType bool
		ok, t := utils.GetFileType(filepath.Ext(name))
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
	_, c.URL = utils.SplitURL(c.URL)

	return resourceGetHandler(c)

}
