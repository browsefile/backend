package config

import (
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
	//uses for external DMZ share request
	Hash string `json:"-"`
}
type AllowedShare struct {
	*UserConfig
	*ShareItem
}

//allow access to the specific share link
func (shr *ShareItem) IsAllowed(user string) (res bool) {
	_, ok := config.GetByUsername(user)

	config.lockR()
	defer config.unlockR()

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
		Hash:          shr.Hash,
	}
	copy(res.AllowUsers, shr.AllowUsers)
	return
}

func (shr *ShareItem) IsActive() (res bool) {
	res = shr != nil && (len(shr.Path) > 0 || len(shr.AllowUsers) > 0 || shr.AllowExternal || shr.AllowLocal)
	return
}

/**
ru  request user su share user
*/
func GetShare(ru, su, reqPath string) (res *ShareItem, user *UserConfig) {
	shareUser, ok := config.GetByUsername(su)

	config.lockR()
	defer config.unlockR()

	if ok {
		item := shareUser.GetShare(reqPath)
		if item != nil && item.IsAllowed(ru) {
			res = item
			user = shareUser
		}
	}

	return
}

//filter out allowed shares, and returns modified path, starting with username
func GetAllowedShares(user string, excludeSelf bool) (res map[string][]*AllowedShare) {
	users := config.GetUsers()

	config.lockR()
	defer config.unlockR()

	isExternal := len(user) == 0
	res = make(map[string][]*AllowedShare)
	//check user and allowed path
	for _, ui := range users {
		for _, shr := range ui.Shares {
			if shr.IsActive() && (isExternal && shr.AllowExternal || !isExternal && shr.AllowLocal || shr.IsAllowed(user)) {
				//ignore own files
				if excludeSelf && strings.EqualFold(ui.Username, user) {
					continue
				}
				if res[ui.Username] == nil {
					res[ui.Username] = make([]*AllowedShare, 0, 10)
				}
				res[ui.Username] = append(res[ui.Username], &AllowedShare{
					ui,
					shr,
				})
			}
		}
	}
	return res
}
