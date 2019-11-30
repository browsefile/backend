package web

import (
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"github.com/maruel/natural"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//generates m3u playlist
func makePlaylist(c *lib.Context) (int, error) {
	var err error

	if len(c.FilePaths) == 0 {
		return http.StatusNotFound, err
	}
	c.RESP.Header().Set("Content-Disposition", "attachment; filename=playlist.m3u")
	c.FitFilter = func(name, p string) bool {
		if ok, t := utils.GetFileType(filepath.Ext(name)); ok && fitMediaFilter(c, t) {
			return true
		}
		return false
	}
	code, err, _ := prepareFiles(c)
	if err != nil {
		return code, err
	}
	sort.Sort(sort.Reverse(byName(c.FilePaths)))
	h := getHost(c)
	for _, p := range c.FilePaths {
		serveFileAsUrl(c, filepath.Base(p), p, h)
	}

	return code, nil
}

func fitMediaFilter(c *lib.Context, t string) bool {
	return c.Audio && t == cnst.AUDIO ||
		c.Video && t == cnst.VIDEO ||
		c.Image && t == cnst.IMAGE
}

//write specific m3u tags into response
func serveFileAsUrl(c *lib.Context, fName, p, host string) {

	io.WriteString(c.RESP, "#EXTINF:0 group-title=\"")
	io.WriteString(c.RESP, "Browsefile")
	io.WriteString(c.RESP, "\",")
	io.WriteString(c.RESP, fName)

	io.WriteString(c.RESP, "\r\n")
	io.WriteString(c.RESP, host)
	io.WriteString(c.RESP, p)
	io.WriteString(c.RESP, "?inline=true")

	if c.IsExternal {
		io.WriteString(c.RESP, "&")
		io.WriteString(c.RESP, cnst.P_EXSHARE)
		io.WriteString(c.RESP, "=1")
	}
	if len(c.Auth) > 0 {
		io.WriteString(c.RESP, "&auth="+c.Auth)
	}

	io.WriteString(c.RESP, "\r\n")

}

//returns correct URL for playlist link in file
func getHost(c *lib.Context) string {
	var h string
	if c.IsExternal {
		h = strings.TrimSuffix(c.Config.ExternalShareHost, "/")
	} else {
		if c.REQ.TLS == nil {
			h = c.Config.Http.IP + ":" + strconv.Itoa(c.Config.Http.Port)
		} else {
			h = c.Config.Tls.IP + ":" + strconv.Itoa(c.Config.Tls.Port)
		}
	}

	if c.IsShare {
		h += "/api/shares/download"
	} else {
		h += "/api/download"
	}
	if !c.IsExternal {
		if c.REQ.TLS != nil {
			h = "https://" + h
		} else {
			h = "http://" + h
		}
	}
	return h
}

type byName []string

// By Name
func (l byName) Len() int {
	return len(l)
}

func (l byName) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Treat upper and lower case equally
func (l byName) Less(i, j int) bool {
	return natural.Less(strings.ToLower(l[j]), strings.ToLower(l[i]))
}
