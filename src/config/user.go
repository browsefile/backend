package config

import (
	"github.com/browsefile/backend/src/cnst"
	"golang.org/x/net/webdav"
	"strings"
)

// User contains the configuration for each user.
type UserConfig struct {
	FirstRun bool `json:"hashPasswordFirstRun"`
	// Tells if this user is an admin.
	Admin bool `json:"admin"`
	// These indicate if the user can perform certain actions.
	AllowEdit bool `json:"allowEdit"` // Edit/rename files
	AllowNew  bool `json:"allowNew"`  // Create files and folders

	// Prevents the user to change its password.
	LockPassword bool `json:"lockPassword"`

	// Locale is the language of the user.
	Locale string `json:"locale"`

	// The hashed password. This never reaches the front-end because it's temporarily
	// emptied during JSON marshall.
	Password string `json:"password"`

	// Username is the user username used to login.
	Username string `json:"username"`

	// User view mode for files and folders.
	ViewMode string `json:"viewMode"`

	Shares []*ShareItem `json:"shares"`
	//authenticate by IP, need to change auth.method
	IpAuth     []string        `json:"ipAuth"`
	DavHandler *webdav.Handler `json:"-"`

	//create files/folders according this ownership
	UID int `json:"uid"`
	GID int `json:"gid"`
}

func (u *UserConfig) copyUser() (res *UserConfig) {
	res = &UserConfig{
		Username:     u.Username,
		FirstRun:     u.FirstRun,
		Password:     u.Password,
		AllowNew:     u.AllowNew,
		LockPassword: u.LockPassword,
		ViewMode:     u.ViewMode,
		Admin:        u.Admin,
		AllowEdit:    u.AllowEdit,
		Locale:       u.Locale,
		UID:          u.UID,
		GID:          u.GID,
		DavHandler:   u.DavHandler,
		IpAuth:       make([]string, len(u.IpAuth)),
	}
	copy(res.IpAuth, u.IpAuth)
	res.Shares = make([]*ShareItem, len(u.Shares))
	for i, uShr := range u.Shares {
		res.Shares[i] = uShr.copyShare()
	}
	return
}
func (u *UserConfig) IsGuest() bool {
	return u.Username == cnst.GUEST

}

func (u *UserConfig) GetShares(relPath string, del bool) (res []*ShareItem) {
	config.lockR()
	defer config.unlockR()
	for _, shr := range u.Shares {
		if del {
			if strings.HasPrefix(relPath, shr.Path) || strings.HasPrefix(shr.Path, relPath) {
				res = append(res, shr.copyShare())
			}
		} else if strings.EqualFold(relPath, shr.Path) {
			res = append(res, shr.copyShare())
			break
		}
	}
	return
}

func (u *UserConfig) deleteShare(relPath string) (res bool) {
	res = false

	for i, shr := range u.Shares {
		if strings.HasPrefix(relPath, shr.Path) || strings.HasPrefix(shr.Path, relPath) {
			u.Shares = append(u.Shares[:i], u.Shares[i+1:]...)
			delSharePath(shr, u.Username)
			res = true
			break
		}

	}

	return
}

func (u *UserConfig) AddShare(shr *ShareItem) (res bool) {
	config.lock()
	shr.Path = strings.TrimSuffix(shr.Path, "/")
	u.Shares = append(u.Shares, shr)
	if shr.AllowExternal {
		shr.Hash = GenShareHash(u.Username, shr.Path)
	}
	res = true
	config.unlock()
	addSharePath(shr, u.Username)
	return
}
