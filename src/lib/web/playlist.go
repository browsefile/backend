package web

import (
	"github.com/browsefile/backend/src/cnst"
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/maruel/natural"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func makePlaylist(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var err error

	if len(c.FilePaths) == 0 {
		return http.StatusNotFound, err
	}
	w.Header().Set("Content-Disposition", "attachment; filename=playlist.m3u")

	paths := fetchFilesRecursively(c)

	sort.Sort(sort.Reverse(byName(paths)))
	h := getHost(c, r)
	for _, p := range paths {
		serveFile(c, w, filepath.Base(p), p, h, c.IsShare)
	}

	return http.StatusOK, nil
}

func fetchFilesRecursively(c *fb.Context) (res []string) {
	var err error
	for _, f := range c.FilePaths {

		if f, err = fileutils.CleanPath(f); err != nil {
			continue
		}

		c.IsRecursive = true
		file, err := fb.MakeInfo(f, f, c)
		if err != nil {
			log.Println(err)
		}
		if file != nil {
			_, paths, err := file.MakeListing(c, func(name, p string) bool {
				if ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(name)); ok && fitMediaFilter(c, t) {
					return true
				}
				return false
			})
			if err == nil {
				res = append(res, paths...)
			}

		}

	}
	return res
}
func fitMediaFilter(c *fb.Context, t string) bool {
	return c.Audio && strings.EqualFold(t, cnst.AUDIO) ||
		c.Video && strings.EqualFold(t, cnst.VIDEO) ||
		c.Image && strings.EqualFold(t, cnst.IMAGE)
}

func serveFile(c *fb.Context, pw http.ResponseWriter, fName, p, host string, isShare bool) {

	io.WriteString(pw, "#EXTINF:0 tvg-name=")
	io.WriteString(pw, fName)
	io.WriteString(pw, "\n")
	io.WriteString(pw, host)
	io.WriteString(pw, p)
	io.WriteString(pw, "?inline=true")

	if c.IsExternalShare() {
		io.WriteString(pw, "&rootHash="+c.RootHash)
	}
	if len(c.Auth) > 0 {
		io.WriteString(pw, "&auth="+c.Auth)
	}

	io.WriteString(pw, "\n\n")

}
func getHost(c *fb.Context, r *http.Request) string {
	var h string
	if c.IsExternalShare() {
		h = strings.TrimSuffix(c.Config.ExternalShareHost, "/")
	} else {
		if r.TLS == nil {
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
	if !c.IsExternalShare() {
		if len(c.Config.TLSKey) > 0 {
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
