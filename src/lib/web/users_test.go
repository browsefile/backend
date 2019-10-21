package web

import (
	"bytes"
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib"
	"net/http"
	"testing"
)

func TestUsersList(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	dat := map[string]interface{}{"u": "/admin", "method": http.MethodPost}

	_, rs, _ := cfg.MakeRequest(cnst.R_USERS, dat, cfg.Usr1, t, false)
	if rs.StatusCode != http.StatusForbidden {
		t.Error("user not allowed to modify other users")
	}
	dat["method"] = http.MethodGet
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.Usr1, t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("user list should be allowed")
	}
	dat["u"] = "/"
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.Usr1, t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("user list should be allowed")
	}
}
func TestUserCreate(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	dat := map[string]interface{}{"u": "/", "method": http.MethodPost}
	//copy user
	usr3, _ := cfg.GetUserByUsername("user1")
	usr3.Username = "user3"
	usr3.Password = "1"
	usr3.FirstRun = true
	buf := new(bytes.Buffer)
	modu := new(ModifyUserRequest)
	modu.What = "user"
	modu.Data = lib.ToUserModel(usr3, cfg.GlobalConfig)
	err := json.NewEncoder(buf).Encode(modu)
	if err != nil {
		t.Error(err)
	}
	dat["body"] = buf

	_, rs, _ := cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, true)

	if rs.StatusCode != http.StatusCreated {
		t.Error("user must be created")
	}

}
func TestUserUpdate(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	dat := map[string]interface{}{"u": "/user1", "method": http.MethodPut}
	//copy user
	buf := new(bytes.Buffer)
	modu := new(ModifyUserRequest)
	modu.What = "user"
	cfg.Usr1.Password = "1"
	modu.Data = lib.ToUserModel(cfg.Usr1, cfg.GlobalConfig)
	err := json.NewEncoder(buf).Encode(modu)
	if err != nil {
		t.Error(err)
	}

	_, rs, _ := cfg.MakeRequest(cnst.R_USERS, dat, cfg.Usr1, t, false)

	if rs.StatusCode != http.StatusForbidden {
		t.Error("user not allowed to create")
	}
	dat["body"] = buf
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, false)

	if rs.StatusCode != http.StatusOK {
		t.Error("user must be created")
	}
	//partial update
	buf.Reset()
	modu.Which = "partial"
	_ = json.NewEncoder(buf).Encode(modu)
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, false)

	if rs.StatusCode != http.StatusOK {
		t.Error("user was not updated partially")
	}
	//update password
	cfg.Usr1.Password = "1"
	modu.Which = "password"
	buf.Reset()
	_ = json.NewEncoder(buf).Encode(modu)
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("user password was no updated")
	}

	//update viewMode
	modu.Which = "viewMode"
	buf.Reset()
	_ = json.NewEncoder(buf).Encode(modu)
	_, rs, _ = cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("user password was no updated")
	}

}
func TestUserDelete(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	dat := map[string]interface{}{"u": "/user1", "method": http.MethodDelete}

	_, rs, _ := cfg.MakeRequest(cnst.R_USERS, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("user not allowed to modify other users")
	}

}
