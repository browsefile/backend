package config

import "strings"

// User contains the configuration for each user.
type UserConfig struct {
	FirstRun bool `json:"firstRun"`
	// Tells if this user is an admin.
	Admin bool `json:"isAdmin"`
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

	// Scope is the path the user has access to.
	Scope string `json:"homePath"`

	// Username is the user username used to login.
	Username string `json:"username" storm:"index,unique"`

	// User view mode for files and folders.
	ViewMode string `json:"viewMode"`
	// system path to store user's image/video previews
	PreviewScope string `json:"previewPath"`
	//enable preview generating by call .sh
	AllowGeneratePreview bool `json:"allowGeneratePreview"`

	Shares []*ShareItem `json:"shares"`
	//authenticate by IP, need to change auth.method
	IpAuth []string `json:"ipAuth"`
}

func (u *UserConfig) copyUser() (res *UserConfig) {
	res = &UserConfig{
		Username:             u.Username,
		FirstRun:             u.FirstRun,
		Password:             u.Password,
		AllowNew:             u.AllowNew,
		LockPassword:         u.LockPassword,
		PreviewScope:         u.PreviewScope,
		Scope:                u.Scope,
		ViewMode:             u.ViewMode,
		Admin:                u.Admin,
		AllowEdit:            u.AllowEdit,
		Locale:               u.Locale,
		AllowGeneratePreview: u.AllowGeneratePreview,
		IpAuth:               make([]string, len(u.IpAuth)),
	}
	copy(res.IpAuth, u.IpAuth)
	res.Shares = make([]*ShareItem, len(u.Shares))
	for i := 0; i < len(u.Shares); i++ {
		res.Shares[i] = u.Shares[i].copyShare()
	}
	return
}

func (u *UserConfig) GetShare(relPath string) (res *ShareItem) {
	for i := 0; i < len(u.Shares); i++ {
		if strings.HasPrefix(u.Shares[i].Path, relPath) {
			res = u.Shares[i].copyShare()
			break
		}

	}
	return
}

func (u *UserConfig) DeleteShare(relPath string) (res bool) {
	config.lock()
	defer config.unlock()
	res = false

	for i := 0; i < len(u.Shares); i++ {
		if strings.HasPrefix(u.Shares[i].Path, relPath) {
			u.Shares = append(u.Shares[:i], u.Shares[i+1:]...)
			res = true
			break
		}

	}
	return
}
func (u *UserConfig) AddShare(shr *ShareItem) (res bool) {
	config.lock()
	defer config.unlock()

	u.Shares = append(u.Shares, shr)
	return
}
