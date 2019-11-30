package web

import (
	"github.com/browsefile/backend/src/cnst"
	"net/http"
	"testing"
)

func TestSearchImageFiles(t *testing.T) {
	searchTest(t, "type:i ", 2, 4, true)
}

func TestSearchByPath(t *testing.T) {
	searchTest(t, "sha", 8, 8, false)
}

func TestSearchImageSharesExternal(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	dat := map[string]interface{}{"query": "type:i "}
	//search in share up
	shr := cfg.Usr1.GetShares(cfg.SharePathUp, false)[0]
	p := shr.ResolveSymlinkName()
	dat[cnst.P_EXSHARE] = shr.Hash
	dat["u"] = "/" + p
	_, rs, _ := cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.Guest, t, true)
	f := *ValidateListingResp(rs, t, 5)
	CheckLink(f, dat, cfg, t, true, true)
}
func TestSearchImageShares(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)

	defer cfg.Clean(t)

	//trying to read sharedeep from admin user
	shr := cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0]
	p := shr.ResolveSymlinkName()

	dat := map[string]interface{}{"u": "/" + cfg.Usr1.Username + "/" + p, "query": "type:i "}
	_, rs, _ := cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.GetAdmin(), t, true)

	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status")
	}
	f := *ValidateListingResp(rs, t, 3)
	CheckLink(f, dat, cfg, t, true, true)

	//search in share up
	shr = cfg.Usr1.GetShares(cfg.SharePathUp, false)[0]
	p = shr.ResolveSymlinkName()
	dat["u"] = "/" + cfg.Usr1.Username + "/" + p
	_, rs, _ = cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.GetAdmin(), t, true)

	f = *ValidateListingResp(rs, t, 5)
	CheckLink(f, dat, cfg, t, true, true)
}
func TestSearchImageShareExternal(t *testing.T) {
	cfg := TServContext{}

	cfg.InitServ(t)
	defer cfg.Clean(t)

	//trying to read sharedeep from admin user
	shr := cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0]
	//for external share we cut first parent, and replace it with root hash
	dat := map[string]interface{}{"u": "/share", "query": "type:i ", cnst.P_EXSHARE: shr.Hash}
	_, rs, _ := cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.GetAdmin(), t, true)
	if rs.StatusCode != http.StatusNotFound {
		t.Errorf("must be 404")
	}

	//search in share up
	shr = cfg.Usr1.GetShares(cfg.SharePathUp, false)[0]
	dat["u"] = "/" + shr.ResolveSymlinkName()
	dat[cnst.P_EXSHARE] = "1"
	_, rs, _ = cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.Guest, t, true)

	f := *ValidateListingResp(rs, t, 5)
	CheckLink(f, dat, cfg, t, true, true)
}

//c1 files count in deep path
//c2 files count in parent path
func searchTest(t *testing.T, q string, c1, c2 int, reqPreview bool) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)

	dat := map[string]interface{}{"u": "/files" + cfg.SharePathDeep, "query": q}
	_, rs, _ := cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.GetAdmin(), t, false)

	f := *ValidateListingResp(rs, t, c1)
	CheckLink(f, dat, cfg, t, false, reqPreview)

	//search in parent
	dat["u"] = "/files" + cfg.SharePathUp
	_, rs, _ = cfg.MakeRequest(cnst.R_SEARCH, dat, cfg.GetAdmin(), t, false)

	f = *ValidateListingResp(rs, t, c2)

	CheckLink(f, dat, cfg, t, false, reqPreview)
}
