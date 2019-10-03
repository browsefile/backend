package config

import (
	"crypto/md5"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
)

//presents 1 share Path in filesystem, and access rules
type ShareItem struct {
	Path string `json:"path"`
	//allow all not registered
	AllowExternal bool `json:"allowExternal"`
	//allow all registered users
	AllowLocal bool `json:"allowLocal"`
	//allowed by only specific users
	AllowUsers []string `json:"allowedUsers"`
	//uses for external DMZ share request
	Hash string `json:"-"`
	//share owner
	User string `json:"-"`
}
type AllowedShare struct {
	*UserConfig
	*ShareItem
}

//allow access to the specific share link
func (shr *ShareItem) IsAllowed(user string) (res bool) {
	usr, ok := config.GetByUsername(user)

	config.lockR()
	defer config.unlockR()

	if ok && shr.AllowLocal && !usr.IsGuest() {
		res = true
	} else if shr.AllowExternal && len(shr.Hash) > 0 {
		res = true
	} else {
		for _, uname := range shr.AllowUsers {
			res = strings.EqualFold(uname, user)
			if res {
				break
			}
		}
	}

	return
}

func (shr *ShareItem) copyShare() (res *ShareItem) {
	res = &ShareItem{
		Path:          shr.Path,
		AllowExternal: shr.AllowExternal,
		AllowLocal:    shr.AllowLocal,
		AllowUsers:    make([]string, len(shr.AllowUsers)),
		Hash:          shr.Hash,
	}
	copy(res.AllowUsers, shr.AllowUsers)
	return
}

func (shr *ShareItem) IsActive() (res bool) {
	res = shr != nil && (len(shr.Path) > 0 || len(shr.AllowUsers) > 0 || shr.AllowExternal || shr.AllowLocal)
	return
}

func addSharePath(shr *ShareItem, own string) {
	if shr.AllowLocal {
		for _, u := range config.Users {
			config.checkShareSymLinkPath(shr, u.Username, own)

		}
	} else if len(shr.AllowUsers) > 0 {
		for _, uName := range shr.AllowUsers {
			u, _ := usersRam[uName]
			config.checkShareSymLinkPath(shr, u.Username, own)
		}
	}
}
func delSharePath(shr *ShareItem, owner string) {
	for _, u := range config.Users {
		dp := filepath.Join(config.FilesPath, u.Username, "shares", owner, shr.Path)
		_ = os.RemoveAll(dp)

	}

}
func GenShareHash(userName, itmPath string) string {
	itmPath = strings.ReplaceAll(itmPath, "/", "")
	return base64.StdEncoding.EncodeToString(md5.New().Sum([]byte(userName + itmPath)))
}
