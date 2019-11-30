package config

import (
	"crypto/md5"
	"encoding/base64"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib/utils"
	"log"
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
}
type AllowedShare struct {
	*UserConfig
	*ShareItem
}

//allow access to the specific share link
func (shr *ShareItem) IsAllowed(user string) (res bool) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	usr, ok := config.GetUserByUsername(user)

	if ok && shr.AllowLocal && !usr.IsGuest() {
		res = true
	} else if shr.AllowExternal && len(shr.Hash) > 0 && user == cnst.GUEST {
		res = true
	} else {
		for _, uname := range shr.AllowUsers {
			res = uname == user
			if res {
				break
			}
		}
	}

	return
}

func (shr *ShareItem) copyShare() (res *ShareItem) {
	updateLock.RLock()
	defer updateLock.RUnlock()
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

func addSharePath(shr *ShareItem, own string) {
	if shr.AllowLocal {
		for _, u := range config.Users {
			processSharePath(shr, u, own)

		}
	} else if len(shr.AllowUsers) > 0 {
		for _, uName := range shr.AllowUsers {
			u, _ := usersRam[uName]
			processSharePath(shr, u, own)
		}
	}
}

//will create share path, or drop share item, in case share not exists
func processSharePath(shr *ShareItem, u *UserConfig, own string) {
	if u.Username == own {
		return
	}
	if !config.checkShareSymLinkPath(shr, u.Username, own) {
		if owner, ok := config.GetUserByUsername(own); ok && owner.DeleteShare(shr.Path) {
			_ = config.Update(owner)
		}
	} else {
		config.checkExternalShareSymLinkPath(shr, own)
	}
}

func delSharePath(shr *ShareItem, owner string) {
	for _, u := range config.Users {
		if owner == u.Username {
			continue
		}
		p := shr.ResolveSymlinkName()

		dp := filepath.Join(config.GetUserSharesPath(u.Username), owner, p)
		_ = os.Remove(dp)
	}
	//drop external share
	dp := filepath.Join(config.GetUserSharexPath(owner), shr.Hash)
	_ = os.Remove(dp)

}
func GenShareHash(userName, itmPath string) string {
	return base64.StdEncoding.EncodeToString(md5.New().Sum([]byte(userName + itmPath)))
}

//returns true in case share good, otherwise original share path does not exists. Will create share if needed
func (cfg *GlobalConfig) checkShareSymLinkPath(shr *ShareItem, consumer, owner string) (res bool) {
	res = true
	dp := filepath.Join(cfg.GetUserSharesPath(consumer), owner)
	//check share exists at shares user dir
	if !createPath(dp) {
		log.Printf("config : Cant create share path at %s ", dp)

	} else {
		//destination path for symlink
		sp := shr.ResolveSymlinkName()
		var err error
		dPath := filepath.Join(dp, sp)
		//source path for symlink
		sPath := filepath.Join(cfg.GetUserHomePath(owner), shr.Path)
		if shr.IsAllowed(consumer) {
			//drop existing share if exists
			_ = os.Remove(dPath)

			//check if share valid
			if _, err = os.Stat(sPath); err != nil && os.IsNotExist(err) {
				log.Printf("config :source '%s' does not exists for %s'", sPath, dPath)
				res = false
			} else if err = os.Symlink(sPath, dPath); err != nil && !os.IsExist(err) {
				//drop not valid symlink, because dav client will fail to read it
				_ = os.Remove(dPath)
				log.Printf("config : Cant create share sym link from '%s' TO '%s'", sPath, dPath)
			}
			if err != nil {
				_ = utils.ModPermission(0, 0, dPath)
			}
		}
	}
	return res
}
func (cfg *GlobalConfig) checkExternalShareSymLinkPath(shr *ShareItem, owner string) (res bool) {
	res = true
	dp := cfg.GetUserSharexPath(owner)
	//check share exists at shares user dir
	sp := shr.ResolveSymlinkName()
	if !createPath(dp) {
		log.Printf("config : Cant create external share path at %s ", dp)

	} else {
		var err error
		dPath := filepath.Join(dp, sp)
		//source path for symlink
		sPath := filepath.Join(cfg.GetUserHomePath(owner), shr.Path)
		//drop existing share if exists
		_ = os.Remove(dPath)

		//check if share valid
		if _, err = os.Stat(sPath); err != nil && os.IsNotExist(err) {
			log.Printf("config :source '%s' does not exists for %s'", sPath, dPath)
			res = false
		} else if err = os.Symlink(sPath, dPath); err != nil && !os.IsExist(err) {
			//drop not valid symlink, because dav client will fail to read it
			_ = os.Remove(dPath)
			log.Printf("config : Cant create share sym link from '%s' TO '%s'", sPath, dPath)
		}
		if err != nil {
			_ = utils.ModPermission(0, 0, dPath)
		}
	}
	return res
}

//will create correct symlink name, err in case hash empty
func (shr *ShareItem) ResolveSymlinkName() (string) {
	d := filepath.Dir(shr.Path)
	return strings.ReplaceAll(strings.TrimPrefix(shr.Path, d), "/", "") + "_" + shr.Hash
}

//take the user from url, find it, after return user preview
func (cfg *GlobalConfig) GetSharePreviewPath(url string, isEx bool) (res, hash string) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	//cut username
	u := strings.TrimPrefix(url, "/")
	if len(u) > 0 {
		arr := strings.Split(u, "/")
		if len(arr) >= 2 && !isEx || isEx && len(arr) >= 1 {
			var ind int
			if isEx {
				ind = 0
			} else {
				ind = 1
			}

			arr2 := strings.Split(arr[ind], "_")
			hash = strings.Split(arr2[len(arr2)-1], "/")[0]
			shr, user := cfg.GetExternal(hash)
			if shr != nil {
				fName := ""
				if len(filepath.Ext(arr[len(arr)-1])) > 0 {
					fName = arr[len(arr)-1]
				}
				res = filepath.Join(cfg.GetUserPreviewPath(user.Username), shr.Path, fName)
			}
		}
	}
	return res, hash
}
