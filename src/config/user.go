package config

import (
	"errors"
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
	updateLock.RLock()
	defer updateLock.RUnlock()
	relPath = strings.TrimSuffix(relPath, "/")
	for _, shr := range u.Shares {
		if del {
			if strings.HasPrefix(relPath, shr.Path) || strings.HasPrefix(shr.Path, relPath) {
				res = append(res, shr.copyShare())
			}
		} else if relPath == shr.Path {
			res = append(res, shr.copyShare())
			break
		}
	}
	return
}

//true in case share deleted
func (u *UserConfig) DeleteShare(relPath string) (res bool) {
	updateLock.RLock()
	defer updateLock.RUnlock()

	return u.deleteShare(relPath)
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
	return res
}

//true in case share added
func (u *UserConfig) AddShare(shr *ShareItem) (res bool) {

	shr.Path = strings.TrimSuffix(shr.Path, "/")
	u.Shares = append(u.Shares, shr)
	shr.Hash = GenShareHash(u.Username, shr.Path)
	res = true
	addSharePath(shr, u.Username)
	return
}

func (cfg *GlobalConfig) GetUserByUsername(username string) (*UserConfig, bool) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	if username == cnst.GUEST {
		admin := cfg.GetAdmin()
		return &UserConfig{
			Username:  username,
			Locale:    admin.Locale,
			Admin:     false,
			ViewMode:  admin.ViewMode,
			AllowNew:  false,
			AllowEdit: false,
		}, true
	}

	res, ok := usersRam[username]
	if !ok {
		return nil, ok
	}

	return res.copyUser(), ok
}
func (cfg *GlobalConfig) GetUserByIp(ip string) (*UserConfig, bool) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	ip = strings.Split(ip, ":")[0]
	res, ok := usersRam[ip]
	if !ok {
		return nil, ok
	}

	return res.copyUser(), ok
}

func (cfg *GlobalConfig) GetUsers() (res []*UserConfig) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	res = make([]*UserConfig, len(cfg.Users))
	for i, u := range cfg.Users {
		res[i] = u.copyUser()
	}

	return res
}

func (cfg *GlobalConfig) AddUser(u *UserConfig) error {
	updateLock.Lock()
	defer updateLock.Unlock()
	_, exists := usersRam[u.Username]
	if exists {
		return errors.New("User exists " + u.Username)
	}

	cfg.Users = append(cfg.Users, u)
	cfg.RefreshUserRam()

	return nil
}
func (cfg *GlobalConfig) UpdatePassword(u *UserConfig) error {
	updateLock.Lock()
	defer updateLock.Unlock()
	i := cfg.getUserIndex(u.Username)
	if i >= 0 {
		//update only specific fields
		cfg.Users[i].Password = u.Password
	} else {
		return errors.New("User does not exists " + u.Username)
	}
	return nil
}
func (cfg *GlobalConfig) Update(u *UserConfig) error {
	updateLock.Lock()
	defer updateLock.Unlock()
	i := cfg.getUserIndex(u.Username)
	if i >= 0 {
		//update only specific fields
		cfg.Users[i].Admin = u.Admin
		cfg.Users[i].ViewMode = u.ViewMode
		cfg.Users[i].FirstRun = u.FirstRun
		cfg.Users[i].Shares = u.Shares
		cfg.Users[i].IpAuth = u.IpAuth
		cfg.Users[i].Locale = u.Locale
		cfg.Users[i].AllowEdit = u.AllowEdit
		cfg.Users[i].AllowNew = u.AllowNew
		cfg.Users[i].LockPassword = u.LockPassword
		cfg.Users[i].UID = u.UID
		cfg.Users[i].GID = u.GID
		cfg.RefreshUserRam()
	} else {
		return errors.New("User does not exists " + u.Username)
	}

	return nil
}

//delete user by username
func (cfg *GlobalConfig) DeleteUser(username string) error {
	updateLock.Lock()
	defer updateLock.Unlock()
	i := cfg.getUserIndex(username)
	if i >= 0 {
		for _, shr := range cfg.Users[i].Shares {
			cfg.Users[i].deleteShare(shr.Path)
		}

		cfg.Users = append(cfg.Users[:i], cfg.Users[i+1:]...)
	}
	cfg.RefreshUserRam()

	return nil
}

//get user index in cfg users array
func (cfg *GlobalConfig) getUserIndex(userName string) int {
	for i, u := range cfg.Users {
		if u.Username == userName {
			return i
		}
	}
	return -1
}
