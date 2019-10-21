package web

import (
	"bytes"
	"github.com/browsefile/backend/src/cnst"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestResourceList(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	dat := map[string]interface{}{"u": "/"}

	_, rs, _ := cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.GetAdmin(), t, false)
	f := *ValidateListingResp(rs, t, 8)
	CheckLink(f, dat, cfg, t, false, true)
}

func TestResourceDelete(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	u := "/test/t.txt"
	dat := map[string]interface{}{"u": u, "method": http.MethodDelete}

	_, rs, _ := cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status ", rs.StatusCode)
	}
	_, err := cfg.AdminFS.Stat(u)
	if os.IsExist(err) {
		t.Error("file", u, "should be deleted")
	}
	//del share
	dat["u"] = cfg.SharePathUp
	_, rs, _ = cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.Usr1, t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status ", rs.StatusCode)
	}
	_, err = cfg.User1FS.Stat(cfg.SharePathUp)
	if os.IsExist(err) {
		t.Error("path", u, "should be deleted")
	}
	cfg.ReadConfigFile()
	usr1, _ := cfg.GetUserByUsername("user1")
	if len(usr1.GetShares(cfg.SharePathUp, false)) > 0 {
		t.Error("share must be deleted as well")

	}

}
func TestResourceMod(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	//check create
	u := "/test/newf/"
	dat := map[string]interface{}{"u": u, "method": http.MethodPost}

	_, rs, _ := cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status ", rs.StatusCode)
	}
	inf, _ := cfg.AdminFS.Stat(u)
	if inf == nil {
		t.Error("path", u, "should be created")
	}
	//check move/rename
	u += "t.png"
	dat["u"] = "/t.png"
	dat["method"] = http.MethodPatch
	dat["destination"] = u
	_, rs, _ = cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status ", rs.StatusCode)
	}
	inf, _ = cfg.AdminFS.Stat(u)
	if inf == nil {
		t.Error("path", u, "should be created")
	}

}
func TestResourceUpload(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	//prepare multipart form with file content
	b := new(bytes.Buffer)
	content := "hello world!"
	b.WriteString(content)
	u := "/n.txt"
	dat := map[string]interface{}{"u": u, "method": http.MethodPut, "body": b,
		"content-type": "text/.txt", "override": "true"}
	_, rs, _ := cfg.MakeRequest(cnst.R_RESOURCE, dat, cfg.GetAdmin(), t, false)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status")
	}
	f, _ := ioutil.ReadFile(cfg.AdminFS.String() + u)
	content2 := string(f)
	if !strings.EqualFold(content, content2) {
		t.Error("file should be created")

	}

}
