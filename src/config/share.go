package config

import (
	"path/filepath"
	"strings"
)

//presents 1 share path in filesystem, and access rules
type ShareItem struct {
	Path string `json:"path"`
	//allow all not registered
	AllowExternal bool `json:"allowExternal"`
	//allow all registered users
	AllowLocal bool `json:"allowLocal"`
	//allowed by only specific users
	AllowUsers []string `json:"allowedUsers"`
}

func (shr *ShareItem) IsAllowed(user string) (res bool) {
	_, ok := config.GetByUsername(user)
	if ok && shr.AllowLocal {
		res = true
	} else if shr.AllowExternal && len(user) == 0 {
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
	}
	copy(res.AllowUsers, shr.AllowUsers)
	return
}

func (shr *ShareItem) IsActive() (res bool) {
	res = shr != nil && (len(shr.Path) > 0 || len(shr.AllowUsers) > 0 || shr.AllowExternal || shr.AllowLocal)
	return
}

/**
ru is request user
 */
func GetShare(ru, su, reqPath string) (res *ShareItem, user *UserConfig) {
	res = new(ShareItem)
	shareUser, ok := config.GetByUsername(su)
	if ok {
		reqPath = strings.TrimSuffix(reqPath, "/")
		item := shareUser.GetShare(reqPath)
		if item != nil && item.IsAllowed(ru) {
			res = item
			user = shareUser
		}
	}

	return
}

//filter out allowed shares
func GetAllowedShares(user string, uNamePath bool) (res []*ShareItem) {
	isExternal := len(user) == 0
	res = make([]*ShareItem, 0, 100)
	//check user and allowed path
	for _, ui := range config.Gets(true) {
		for _, shr := range ui.Shares {
			if shr.IsActive() && (isExternal && shr.AllowExternal || !isExternal && shr.AllowLocal || shr.IsAllowed(user)) {
				res = append(res, shr)
				if uNamePath {
					shr.Path = filepath.Join(ui.Username, shr.Path)
				}
			}
		}
	}
	return res
}
