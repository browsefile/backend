package config

import (
	"log"
	"strings"
	"testing"
)

func TestUserMod(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)
	var err error
	if i := cfg.getUserIndex("admin"); i > 0 {
		t.Fatal(err)
	}
	if _, ok := cfg.GetUserByUsername("admin"); !ok {
		t.Error()
		t.Fatal("cant get admin")
	}
	if cfg.GetAdmin() == nil {
		t.Fatal("can't get admin user")
	}
	if err = cfg.AddUser(cfg.Usr1); err == nil {
		t.Fatal("not fail on existing user")
	}
	if len(cfg.GetUsers()) != 3 || len(cfg.Users) != 3 || len(usersRam) != 3 {
		t.Fatal("wrong len of users")
	}
	//test update
	cfg.Usr1 = cfg.Usr1.copyUser()
	cfg.Usr1.Locale = "fr"
	err = cfg.Update(cfg.Usr1)
	cfg.Usr1, _ = cfg.GetUserByUsername(cfg.Usr1.Username)
	if err != nil || !strings.EqualFold(cfg.Usr1.Locale, "fr") {
		t.Fatal("user update fail")
	}
	//test delete
	_ = cfg.DeleteUser("user1")

	if len(cfg.GetUsers()) != 2 || len(cfg.Users) != 2 || len(usersRam) != 2 {
		t.Fatal("wrong len of users")
	}

	_, ok := cfg.GetUserByUsername("user2")

	if !ok {
		t.Fatal("user not exists")
	}
}
func TestUser(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	admin, _ := cfg.GetUserByUsername("admin")
	if admin.IsGuest() {
		t.Fatal("admin not guest")
	}
	admin.Username = "guest"
	if !admin.IsGuest() {
		t.Fatal("guest check fail")
	}

}
func TestUserAuthByIp(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	admin, _ := cfg.GetUserByUsername("admin")

	admin.IpAuth = []string{"127.0.0.1"}
	_ = cfg.Update(admin)
	cfg.WriteConfig()
	cfg2 := GlobalConfig{Path: cfg.Path, FilesPath: cfg.FilesPath}
	cfg2.ReadConfigFile()
	if _, ok := cfg2.GetUserByIp("127.0.0.1"); !ok {
		t.Fatal("cant fetch user ")
	}
}
func TestUpdatePassword(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	var err error
	admin, _ := cfg.GetUserByUsername("admin")
	admin.Password = "1"
	if err = cfg.UpdatePassword(admin); err != nil {
		log.Println(err)
	}
	admin, _ = cfg.GetUserByUsername("admin")
	if !strings.EqualFold("1", admin.Password) {
		t.Fatal("Can't upd password")
	}

}

func TestGuestUser(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	g, _ := cfg.GetUserByUsername("guest")
	if !g.IsGuest() {
		t.Fatal("user not guest, but should be")
	}
	//check guest permissions
	if g.AllowEdit || g.AllowNew {
		t.Fatal("guest can't write to files path")
	}

}
func TestNotExisting(t *testing.T) {
	anon := "vasya"
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	_, ok := cfg.GetUserByUsername(anon)
	if ok {
		t.Fatal("user should not exists")
	}
	if cfg.getUserIndex(anon) != -1 {
		t.Fatal("user index should not exists")
	}
	usr := cfg.MakeUser("usr")
	err := cfg.UpdatePassword(usr)
	if err == nil {
		t.Fatal("user should not exists")
	}
	err = cfg.Update(usr)
	if err == nil {
		t.Fatal("user should not exists")
	}
	_, ok = cfg.GetUserByIp("127.0.0.1")
	if ok {
		t.Fatal("user should not exists")
	}

}

func TestAddShare(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	defer cfg.Clean(t)
	if len(cfg.Usr1.Shares) != 0 {
		l := len(cfg.Usr1.Shares)
		t.Logf("shares amount %d", l)
		t.Error("share must be 0")
	}
	shrUp := &ShareItem{Path: cfg.SharePathUp, AllowLocal: true, AllowExternal: true}
	cfg.Usr1.AddShare(shrUp)
	//_ = cfg.Update(cfg.Usr1)
	//cfg.Usr1, _ = cfg.GetUserByUsername("user1")

	//add shares to user1
	shrDeep := &ShareItem{Path: cfg.SharePathDeep, AllowUsers: []string{"admin"}}
	cfg.Usr1.AddShare(shrDeep)
	_ = cfg.Update(cfg.Usr1)
	cfg.Usr1.DeleteShare(cfg.SharePathUp)
	_ = cfg.Update(cfg.Usr1)
	if len(cfg.Usr1.Shares) != 1 {
		l := len(cfg.Usr1.Shares)
		t.Logf("shares amount %d", l)
		t.Error("share must be deleted")
	}

}
