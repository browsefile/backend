package web

import "C"
import (
	fb "github.com/browsefile/backend/src/lib"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func fixNonStandardURIEnc(p string) (rs string) {
	//yeah, some browsers unescape, someone escape, whatever
	if strings.Contains(p, "%") {
		var err error
		rs, err = url.QueryUnescape(p)
		if err != nil {
			log.Println(err)
		}

	} else if strings.Contains(p, " ") {
		rs = url.QueryEscape(p)
	} else {
		rs = p
	}
	return rs
}

// set router, and all other params to the context, returns true in case request are about shares
func processParams(c *fb.Context, r *http.Request) (isShares bool) {

	if c.Query == nil {
		c.Query = r.URL.Query()
	}
	c.Sort = c.Query.Get("sort")
	c.Order = c.Query.Get("order")
	c.PreviewType = c.Query.Get("previewType")
	c.Inline, _ = strconv.ParseBool(c.Query.Get("inline"))
	c.RootHash = c.Query.Get("rootHash")
	c.Checksum = c.Query.Get("checksum")
	c.ShareType = c.Query.Get("share")
	c.IsRecursive, _ = strconv.ParseBool(c.Query.Get("recursive"))
	c.Override, _ = strconv.ParseBool(c.Query.Get("override"))
	c.Algo = c.Query.Get("algo")
	c.Auth = c.Query.Get("auth")
	c.RootHash = fixNonStandardURIEnc(c.RootHash)
	//search request
	q := c.Query.Get("query")
	if len(q) > 0 {
		if strings.Contains(q, "type") {
			arr := strings.Split(q, ":")
			arr = strings.Split(arr[1], " ")
			c.SearchString = arr[1]
			setFileType(c, arr[0])
		} else {
			c.SearchString = q
		}
		//c.Query.Del("query")
	}

	isShares = setRouter(c, r)

	if len(c.Algo) > 0 && !strings.HasPrefix(c.Algo, "z") {
		arr := strings.Split(c.Algo, "_")
		if len(arr) > 1 {
			setFileType(c, arr[1])
		}
		if (c.Image || c.Audio || c.Video) && strings.EqualFold(arr[0], "m3u") {
			c.Algo = arr[0]
			if isShares {
				c.Router = "playlist-share"
			} else {
				c.Router = "playlist"
			}

		}
	}

	//in case download, might be multiple files
	if strings.HasPrefix(c.Router, "dow") || strings.HasPrefix(c.Router, "pla") {
		f := c.Query.Get("files")
		if len(f) > 0 {
			c.FilePaths = strings.Split(f, ",")
		}

	} else if r.Method == http.MethodPatch {
		c.Destination = r.Header.Get("Destination")
		if len(c.Destination) == 0 {
			c.Destination = c.Query.Get("destination")
		}
		c.Action = r.Header.Get("action")
		if len(c.Action) == 0 {
			c.Action = c.Query.Get("action")
		}
	}
	r.URL.RawQuery = ""

	return

}
func setFileType(c *fb.Context, t string) {
	c.Image = strings.Contains(t, "i")
	c.Audio = strings.Contains(t, "a")
	c.Video = strings.Contains(t, "v")
	c.Pdf = strings.Contains(t, "p")
}

func setRouter(c *fb.Context, r *http.Request) (isShares bool) {
	c.Router, r.URL.Path = splitURL(r.URL.Path)
	if strings.EqualFold(c.Router, "search") {
		r, _ := splitURL(r.URL.Path)
		isShares = strings.HasPrefix(r, "shares")
	} else {
		isShares = strings.HasPrefix(c.Router, "shares")
	}
	//redirect to the real handler in shares case
	if isShares {
		//possibility to process shares view/download
		if !(strings.EqualFold(c.ShareType, "my-list") ||
			strings.EqualFold(c.ShareType, "my") ||
			strings.EqualFold(c.ShareType, "list") ||
			strings.EqualFold(c.ShareType, "gen-ex")) {
			c.Router, r.URL.Path = splitURL(r.URL.Path)
		}

		if c.Router == "download" {
			c.Router = "download-share"
		} else if c.Router == "resource" || c.Router == "external" {
			c.Router = "shares"
		}
	}

	return
}

// splitURL splits the path and returns everything that stands
// before the first slash and everything that goes after.
func splitURL(path string) (string, string) {
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
