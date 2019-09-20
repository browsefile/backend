package web

import (
	fb "github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/maruel/natural"
	"io"
	"net/http"
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
	for _, p := range paths {
		serveFile(c, w, filepath.Base(p), p)
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
		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			//if !info.IsDir() && err != nil {

				if ok, t := fileutils.GetBasedOnExtensions(filepath.Ext(info.Name())); ok && fitMediaFilter(c, t) {
					res = append(res, strings.Replace(path, c.GetUserHomePath(), "", -1))
				}

			//}
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
	for _, p := range *items {
		c.User.Username = p.User
		c.FilePaths = []string{p.Path}
		for _, fp := range fetchFilesRecursively(c, false) {
			if c.IsExternalShare() {
				fp += "?rootHash=" + p.Hash
			}
			if len(p.User) > 0 {
				s := "?"
				if c.IsExternalShare() {
					s = "&"
				}
				fp += s + "share=" + p.User
			}

			paths = append(paths, fp)

		}
	}

	sort.Sort(sort.Reverse(byName(paths)))
	for _, p := range paths {
		serveFile(c, w, "", p)
	}
	return http.StatusOK, nil
}
func fitMediaFilter(c *fb.Context, t string) bool {
	return c.Audio && strings.EqualFold(t, "audio") ||
		c.Video && strings.EqualFold(t, "video") ||
		c.Image && strings.EqualFold(t, "image")
}

func serveFile(c *fb.Context, pw http.ResponseWriter, fName, p string) {

	io.WriteString(pw, "#EXTINF:0 tvg-name=")
	io.WriteString(pw, fName)
	io.WriteString(pw, "\n")
	io.WriteString(pw, getHost(c))
	io.WriteString(pw, p)
	if c.IsExternalShare() {
		io.WriteString(pw, "&inline=true")
	} else {
		io.WriteString(pw, "?inline=true")
	}

	if len(c.Auth) > 0 {
		io.WriteString(pw, "&auth="+c.Auth)
	}

	io.WriteString(pw, "\n\n")

}
func getHost(c *fb.Context) string {
	h := c.Config.IP + ":" + strconv.Itoa(c.Config.Port) + "/api/download"
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
