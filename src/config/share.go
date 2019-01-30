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
		for i := 0; i < len(shr.AllowUsers); i++ {
			res = strings.EqualFold(shr.AllowUsers[i], user)
			if res {
				break
			}
		}
	}

	return
}

func (shr *ShareItem) ValidPath(path string) (res bool) {
	res = false
	rp := strings.Split(path, "/")
	s := strings.Split(shr.Path, "/")
	sLen := len(s)

	if len(rp) >= sLen {
		var c int
		for i := 0; i < sLen; i++ {
			if strings.EqualFold(s[i], rp[i]) {
				c++
			}
		}
		res = c == sLen
	}

	return res

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

func GetShare(ru, path string) (res *ShareItem, user *UserConfig) {
	res = new(ShareItem)
	sUname := strings.Split(path, "/")[1]
	shareUser, ok := config.GetByUsername(sUname)
	if ok {
		path = strings.Replace(path, "/"+sUname, "", 1)
		item := shareUser.GetShare(path)
		if item != nil && item.IsAllowed(ru) && item.ValidPath(path) {
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
	users := config.Gets(true)

	//check user and allowed path
	for i := 0; i < len(users); i++ {
		for j := 0; j < len(users[i].Shares); j++ {
			shr := users[i].Shares[j]
			if shr.IsActive() && (isExternal && shr.AllowExternal || !isExternal && shr.AllowLocal || shr.IsAllowed(user)) {
				res = append(res, shr)
				if uNamePath {
					shr.Path = filepath.Join(users[i].Username, shr.Path)
				}
			}

		}
	}
	return res
}
