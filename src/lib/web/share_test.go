package web

import (
	"bytes"
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSharePreviewPath(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	p := cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0].ResolveSymlinkName()
	dat := map[string]interface{}{"u": "/user1/" + p, "share": "list"}

	_, rs, _ := cfg.MakeRequest(cnst.R_SHARES, dat, cfg.GetAdmin(), t, true)
	f := *ValidateListingResp(rs, t, 8)
	CheckLink(f, dat, cfg, t, true, true)
	_, err := os.Stat(cfg.PreviewConf.ScriptPath)
	if err != nil {
		//wait until image will be there
		runtime.Gosched()
		time.Sleep(500 * time.Millisecond)
		_, err = cfg.User1FSPreview.Stat(filepath.Join(cfg.SharePathDeep, "real.jpg"))
		if err != nil {
			t.Error("preview should be generated in correct path at", filepath.Join(cfg.User1FSPreview.String(), cfg.SharePathDeep, "real.jpg"))
		}
	}
}
func TestShareMeta(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	dat := map[string]interface{}{"u": cfg.SharePathDeep, "share": "my-meta"}
	_, rs, _ := cfg.MakeRequest(cnst.R_SHARES, dat, cfg.GetAdmin(), t, true)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong listing status at link :", rs.Request.URL.String())
	}
	itm := &config.ShareItem{}

	err := json.NewDecoder(rs.Body).Decode(itm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(itm.Path, cfg.SharePathDeep) {
		t.Error("share path must be same")
	}

}
func TestShareGenEx(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	dat := map[string]interface{}{"u": cfg.SharePathDeep, "share": "gen-ex", "method": http.MethodPost}
	_, rs, _ := cfg.MakeRequest(cnst.R_SHARES, dat, cfg.Usr1, t, true)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong listing status at link :", rs.Request.URL.String())
	}

	b, _ := ioutil.ReadAll(rs.Body)
	link := string(b)
	link, _ = url.QueryUnescape(link)
	if !strings.Contains(link, cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0].Hash) {
		t.Error("share path must be same")
	}

}
func TestShareCreate(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	p := "/ne"

	newShr := config.ShareItem{}
	newShr.Path = p
	newShr.AllowExternal = true
	newShr.AllowLocal = true
	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(newShr)
	_ = cfg.AdminFS.Mkdir(p, cnst.PERM_DEFAULT, 0, 0)
	dat := map[string]interface{}{"u": "/", "share": "my-meta", "method": http.MethodPost, "body": buf}
	_, rs, _ := cfg.MakeRequest(cnst.R_SHARES, dat, cfg.GetAdmin(), t, true)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong listing status at link :", rs.Request.URL.String())
	}
	err := json.NewDecoder(rs.Body).Decode(&newShr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(newShr.Path, p) {
		t.Error("path must be the same")
	}
}
func TestShareDelete(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	dat := map[string]interface{}{"u": cfg.SharePathDeep, "share": "my-meta", "method": http.MethodDelete}
	_, rs, _ := cfg.MakeRequest(cnst.R_SHARES, dat, cfg.Usr1, t, true)
	if rs.StatusCode != http.StatusOK {
		t.Error("wrong listing status at link :", rs.Request.URL.String())
	}
}
