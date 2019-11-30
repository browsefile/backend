package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// set router, and all other params to the context, returns true in case request are about shares
func ProcessParams(c *fb.Context) (isShares bool) {

	if c.Query == nil {
		c.Query = c.REQ.URL.Query()
	}

	c.Sort = c.Query.Get("sort")
	c.Order = c.Query.Get("order")
	c.PreviewType = c.Query.Get(cnst.P_PREVIEW_TYPE)
	c.Inline, _ = strconv.ParseBool(c.Query.Get("inline"))
	c.IsExternal = len(c.Query.Get(cnst.P_EXSHARE)) > 0
	c.Checksum = c.Query.Get("checksum")
	c.ShareType = c.Query.Get("share")
	c.IsRecursive, _ = strconv.ParseBool(c.Query.Get("recursive"))
	c.Override, _ = strconv.ParseBool(c.Query.Get("override"))
	c.Algo = c.Query.Get("algo")
	c.Auth = c.Query.Get("auth")
	if len(c.Auth) == 0 {
		c.Auth = c.REQ.Header.Get(cnst.H_XAUTH)
	}
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

	isShares = setRouter(c)

	if len(c.Algo) > 0 && !strings.HasPrefix(c.Algo, "zip") {
		arr := strings.Split(c.Algo, "_")
		if len(arr) > 1 {
			setFileType(c, arr[1])
		}
		if (c.Image || c.Audio || c.Video) && arr[0] == "m3u" {
			c.Algo = arr[0]
			c.Router = cnst.R_PLAYLIST
		}
	}

	//in case download, might be multiple files
	if cnst.R_DOWNLOAD == c.Router ||
		cnst.R_PLAYLIST == c.Router {
		f := c.Query.Get("files")
		if len(f) > 0 {
			c.FilePaths = strings.Split(f, ",")
			for i, file := range c.FilePaths {
				c.FilePaths [i], _ = url.PathUnescape(file)

			}
		}
	} else if c.REQ.Method == http.MethodPatch {
		c.Destination = c.REQ.Header.Get("Destination")
		if len(c.Destination) == 0 {
			c.Destination = c.Query.Get("destination")
		}
		c.Action = c.REQ.Header.Get("action")
		if len(c.Action) == 0 {
			c.Action = c.Query.Get("action")
		}
	}
	c.URL = c.REQ.URL.Path
	c.Method = c.REQ.Method

	c.REQ.URL.RawQuery = ""

	return

}
func setFileType(c *fb.Context, t string) {
	c.Image = strings.Contains(t, "i")
	c.Audio = strings.Contains(t, "a")
	c.Video = strings.Contains(t, "v")
	c.Pdf = strings.Contains(t, "p")
}

func setRouter(c *fb.Context) (isShares bool) {
	c.Router, c.REQ.URL.Path = utils.SplitURL(c.REQ.URL.Path)
	if c.Router == cnst.R_SEARCH {
		r, _ := utils.SplitURL(c.REQ.URL.Path)
		isShares = r == cnst.R_SHARES
	} else {
		isShares = c.Router == cnst.R_SHARES
	}
	c.Params.IsShare = isShares
	if isShares {
		rp, p := utils.SplitURL(c.REQ.URL.Path)
		if rp == cnst.R_DOWNLOAD {
			c.Router = cnst.R_DOWNLOAD
			c.REQ.URL.Path = p
		}
	}
	//redirect to the real handler in shares case
	if isShares && (c.Router == cnst.R_RESOURCE) {
		c.Router = cnst.R_SHARES
	}

	return
}
