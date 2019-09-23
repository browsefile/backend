package web

import (
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/maruel/natural"
	"io"
	"net/http"
	"net/url"
	"os"
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

	paths := fetchFilesRecursively(c, true)

	sort.Sort(sort.Reverse(byName(paths)))
	h := getHost(c, false)
	for _, p := range paths {
		serveFile(c, w, filepath.Base(p), p, h, false)
	}

	return http.StatusOK, nil
}

func fetchFilesRecursively(c *fb.Context, joinHome bool) []string {
	var err error
	var res []string
	for _, f := range c.FilePaths {

		if f, err = fileutils.CleanPath(f); err != nil {
			continue
		}

		var p string
		if joinHome {
			p = filepath.Join(c.GetUserHomePath(), f)
		} else {
			p = f
		}
		itm, _ := getShare("", c)

		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(info.Name())); ok && fitMediaFilter(c, t) {

				// if request to generate external share, we have to cut share item path, since rootHash replaces it
				// and have to deal with path replacement, if we reuse download component, because still need absolute path in order to walk on it
				if !joinHome && len(c.RootHash) > 0 {

					res = append(res, strings.TrimPrefix(path, c.GetUserHomePath()+itm.Path))
				} else {
					res = append(res, strings.TrimPrefix(path, c.GetUserHomePath()))
				}

			}

			return err
		})
	}
	return res
}
func makeSharePlaylist(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var err error
	if len(c.FilePaths) == 0 {
		return http.StatusNotFound, err
	}

	items, err := genSharePaths(c)
	if err != nil {
		return http.StatusNotFound, err
	}
	w.Header().Set("Content-Disposition", "attachment; filename=playlist.m3u")

	var paths []string
	h := getHost(c, true)
	for _, p := range *items {
		c.User.Username = p.User
		c.FilePaths = []string{p.Path}
		c.RootHash = p.Hash
		for _, fp := range fetchFilesRecursively(c, false) {
			if c.IsExternalShare() {
				fp += "?rootHash=" + url.QueryEscape(p.Hash)
			} else {
				fp += "?share=" + p.User
			}

			paths = append(paths, fp)

		}
	}

	sort.Sort(sort.Reverse(byName(paths)))
	for _, p := range paths {
		serveFile(c, w, "", p, h, true)
	}
	return http.StatusOK, nil
}
func fitMediaFilter(c *fb.Context, t string) bool {
	return c.Audio && strings.EqualFold(t, "audio") ||
		c.Video && strings.EqualFold(t, "video") ||
		c.Image && strings.EqualFold(t, "image")
}

func serveFile(c *fb.Context, pw http.ResponseWriter, fName, p, host string, isShare bool) {

	io.WriteString(pw, "#EXTINF:0 tvg-name=")
	io.WriteString(pw, fName)
	io.WriteString(pw, "\n")
	io.WriteString(pw, host)
	io.WriteString(pw, p)
	if isShare {
		io.WriteString(pw, "&inline=true")
	} else {
		io.WriteString(pw, "?inline=true")
	}

	if len(c.Auth) > 0 {
		io.WriteString(pw, "&auth="+c.Auth)
	}

	io.WriteString(pw, "\n\n")

}
func getHost(c *fb.Context, isShare bool) string {
	h := c.Config.IP + ":" + strconv.Itoa(c.Config.Port)
	if isShare {
		h += "/api/shares/download"
	} else {
		h += "/api/download"
	}
	if len(c.Config.TLSKey) > 0 {
		h = "https://" + h
	} else {
		h = "http://" + h
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
