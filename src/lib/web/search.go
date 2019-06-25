package web

import (
	fb "github.com/browsefile/backend/src/lib"
	"net/http"
	"strings"
)

func searchHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {

	c.IsRecursive = true
	return resourceGetHandler(c, w, r, func(f *fb.File) bool {
		fitUrl := strings.Contains(strings.ToLower(f.URL), strings.ToLower(c.SearchString))
		fitType := strings.EqualFold(f.Type, c.SearchType)
		hasType := len(c.SearchType) > 0

		return hasType && fitType && fitUrl || !hasType && fitUrl
	})
}
