package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharePathMod(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)

	var err error

	//user1 share to all local users
	shrUp := &ShareItem{Path: cfg.SharePathUp, AllowLocal: true, AllowExternal: true}
	cfg.Usr1.AddShare(shrUp)

	//add shares to user1
	shrDeep := &ShareItem{Path: cfg.SharePathDeep, AllowUsers: []string{"admin"}}
	cfg.Usr1.AddShare(shrDeep)
	_ = cfg.Update(cfg.Usr1)
	cfg.WriteConfig()
	cfg.ReadConfigFile()

	//validate share path for user owner user1
	_, err = cfg.User1FS.Stat(cfg.SharePathDeep)
	//os.Remove resolve link, and delete share in actual user's real files
	if err != nil {
		t.Fatal("share path does not exists, but should be", err)
	}
	err = cfg.Update(cfg.Usr1)

	if err != nil {
		t.Fatal(err)
	}
	//validate share path for user owner user1
	_, err = cfg.User1FS.Stat(cfg.SharePathDeep)
	if err != nil || os.IsNotExist(err) {
		t.Fatal("share path does not exists, but should be", err)
	}
	//test parent share present at user2
	p := shrUp.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err != nil {
		t.Fatal("parent share does not exists, but should be", err)
	}
	//test child share present at user2
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("share does not exists, but should be", err)
	}
	//test child share present at admin
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.AdminFSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err != nil {
		t.Fatal("share does not exists, but should be", err)
	}

	cfg.Usr1, _ = cfg.GetUserByUsername("user1")
	shrDeep = cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0]
	ok := cfg.Usr1.DeleteShare(cfg.SharePathUp)
	if !ok {
		t.Fatal("parent share was not deleted")
	}
	ok = cfg.Usr1.DeleteShare(cfg.SharePathDeep)
	if !ok {
		t.Fatal("share was not deleted")
	}
	_ = cfg.Update(cfg.Usr1)
	if len(cfg.Usr1.Shares) > 0 {
		t.Fatal("shares should be deleted")
	}

	//test parent share present at user2
	p = shrUp.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("parent share exists, but should not be", err)
	}
	//test child share present at user2
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("share exists, but should not be", err)
	}
	//test child share present at admin
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.AdminFSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("share exists, but should not be", err)
	}

	_, err = cfg.User1FS.Stat(cfg.SharePathDeep)
	if err != nil || os.IsNotExist(err) {
		t.Fatal("share path does not exists, but should be", err)
	}
	//trying to modify upper share in path for user1
	shrDeep = &ShareItem{AllowUsers: []string{"user2"}, Path: cfg.SharePathUp}
	cfg.Usr1.AddShare(shrDeep)
	_ = cfg.Update(cfg.Usr1)
	cfg.WriteConfig()
	cfg.ReadConfigFile()
	ok = cfg.Usr1.DeleteShare(shrDeep.Path)
	if !ok {
		t.Fatal("share was not deleted")
	}

	//validate share path for user owner user1
	_, err = cfg.User1FS.Stat(cfg.SharePathDeep)
	//os.Remove resolve link, and delete share in actual user's real files
	if err != nil {
		t.Fatal("share path does not exists, but should be", err)
	}

}
func TestResolveSymLinkPath(t *testing.T) {
	shr := &ShareItem{Path: "/path/to/dir", AllowLocal: true, AllowExternal: true, Hash: "0x123"}
	p := shr.ResolveSymlinkName()
	if !strings.HasPrefix(p, "dir_") {
		t.Fatal("symlink should start with path name")
	}
	if !strings.HasSuffix(p, shr.Hash) {
		t.Fatal("symlink should ends with share hash")
	}

}

func TestDeleteUserAndShares(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)

	var err error
	//user1 share to all local users
	shrUp := &ShareItem{Path: cfg.SharePathUp, AllowLocal: true, AllowExternal: true}
	cfg.Usr1.AddShare(shrUp)

	//add shares to user1
	shrDeep := &ShareItem{Path: cfg.SharePathDeep, AllowUsers: []string{"admin"}}
	cfg.Usr1.AddShare(shrDeep)
	_ = cfg.Update(cfg.Usr1)
	cfg.WriteConfig()
	cfg.ReadConfigFile()

	if len(cfg.Usr1.GetShares(cfg.SharePathUp, true)) != 2 {
		t.Fatal("all child path shares must be present")
	}

	_ = cfg.DeleteUser(cfg.Usr1.Username)
	//test parent share present at user2
	p := shrUp.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("parent share exists, but should not be", err)
	}
	//test child share present at user2
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.User2FSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("share exists, but should not be", err)
	}
	//test child share present at admin
	p = shrDeep.ResolveSymlinkName()
	_, err = cfg.AdminFSShare.Stat(filepath.Join(cfg.Usr1.Username, p))
	if err == nil {
		t.Fatal("share exists, but should not be", err)
	}

	_, err = cfg.User1FS.Stat(cfg.SharePathDeep)
	if err != nil || os.IsNotExist(err) {
		t.Fatal("share path does not exists, but should be", err)
	}

}
func TestExternal(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)

	cfg.SharePathUp = "/test"
	//create users
	shrUp := &ShareItem{Path: cfg.SharePathUp, AllowLocal: false, AllowExternal: true}
	cfg.Usr1.AddShare(shrUp)
	_ = cfg.Update(cfg.Usr1)

	shrUp, cfg.Usr1 = cfg.GetExternal(shrUp.Hash)
	if shrUp == nil || cfg.Usr1 == nil {
		t.Fatal("should find at least 1 external share by hash")
	}
	if !shrUp.IsAllowed("guest") {
		t.Fatal("external share should be allowed to the guest")
	}
}
func TestSharePreviewPath(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)

	shrUp := &ShareItem{Path: "/so/path", AllowLocal: false, AllowExternal: false}
	cfg.Usr1.AddShare(shrUp)
	_ = cfg.Update(cfg.Usr1)
	p := shrUp.ResolveSymlinkName()
	p, _ = cfg.GetSharePreviewPath("/user1/"+p+"/path", false)
	if !strings.HasSuffix(p, "preview/so/path") {
		t.Fatal("wrong preview path for share consumer")
	}
	p, h := cfg.GetSharePreviewPath(shrUp.ResolveSymlinkName()+"/path", true)
	if !strings.HasSuffix(p, "preview/so/path") || !strings.EqualFold(shrUp.Hash, h) {
		t.Fatal("wrong ex share hash")
	}
}

/*func TestShareSymlink(t *testing.T) {
	cfg := GetDefConfig()
	defer Clean(cfg, t)
	cfg.SharePathUp := "/test"
	//create users
	cfg.Usr1 := makeUser("user1")
	_ = cfg.AddUser(cfg.Usr1)
	cfg.Usr1, _ = cfg.GetUserByUsername("user1")
	shrUp := &ShareItem{Path: cfg.SharePathUp, AllowLocal: true, AllowExternal: false}
	cfg.Usr1.AddShare(shrUp)

	_ = cfg.Update(cfg.Usr1)
	cfg.checkShareSymLinkPath = func(shr *ShareItem, consumer, owner string) (res bool) {
		return false
	}
	processSharePath(shrUp, cfg.GetAdmin(), cfg.Usr1.Username)
}
*/
