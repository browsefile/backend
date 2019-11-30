package web

import (
	"github.com/browsefile/backend/src/cnst"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestPlaylistOnFilesDir(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	testPlaylistOnDir(&cfg, t, false, map[string]interface{}{
		"u": "/", "files": []string{cfg.SharePathDeep}}, 4)
}
func TestPlaylistOnFile(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	testPlaylistOnDir(&cfg, t, false, map[string]interface{}{"u": "/", "files": []string{cfg.SharePathDeep + "/t.png"}}, 1)
}
func TestPlaylistOnShareFile(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	l := cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0].ResolveSymlinkName()
	p := cfg.Usr1.Username + "/" + l + "/t.png"

	testPlaylistOnDir(&cfg, t, true, map[string]interface{}{"u": "/", "files": []string{p}}, 1)
}
func TestPlaylistOnShareDir(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	l := cfg.Usr1.GetShares(cfg.SharePathDeep, false)[0].ResolveSymlinkName()
	p := cfg.Usr1.Username + "/" + l

	testPlaylistOnDir(&cfg, t, true, map[string]interface{}{"u": "/", "files": []string{p}}, 5)
}
func TestPlaylistOnExternalShareDir(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	shr := cfg.Usr1.GetShares(cfg.SharePathUp, false)[0]
	l := shr.ResolveSymlinkName()
	p := "/" + l

	testPlaylistOnDir(&cfg, t, true, map[string]interface{}{cnst.P_EXSHARE: "1", "u": p, "files": []string{p}}, 9)
}
func TestPlaylistOnExternalShareDirParent(t *testing.T) {
	cfg := TServContext{}
	cfg.InitServ(t)
	defer cfg.Clean(t)
	shr := cfg.Usr1.GetShares(cfg.SharePathUp, false)[0]
	l := shr.ResolveSymlinkName()
	p := "/" + l

	testPlaylistOnDir(&cfg, t, true, map[string]interface{}{cnst.P_EXSHARE: shr.Hash, "u": p, "files": []string{p}}, 9)
}

func testPlaylistOnDir(cfg *TServContext, t *testing.T, isShare bool, data map[string]interface{}, lCount int) {
	_, rs, _ := cfg.MakeRequest(cnst.R_PLAYLIST, data, cfg.GetAdmin(), t, isShare)

	if rs.StatusCode != http.StatusOK {
		t.Error("wrong status")
	}

	byteRS, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		t.Error(err)
	}
	if len(byteRS) == 0 {
		t.Error("response must be present")
	}

	validatePlaylistLinks(cfg, string(byteRS), t, lCount)
}

func validatePlaylistLinks(cfg *TServContext, links string, t *testing.T, count int) {
	count *= 2
	arr := strings.Split(links, "\n")
	arr = arr[:len(arr)-1]
	if len(arr) != count {
		t.Error(links)
		t.Fatal("wrong links count, expected ", count, "actual", len(arr))
	}
	_, p, _ := net.SplitHostPort(cfg.Srv.Listener.Addr().String())
	for i := 1; i < len(arr); i += 2 {
		if len(arr[i]) > 0 {
			arr[i] = strings.ReplaceAll(arr[i], "\r", "")
			arr[i] = strings.ReplaceAll(arr[i], strconv.Itoa(cfg.GlobalConfig.Http.Port), p)
			if !cfg.ValidateDownloadLink(arr[i], t) {
				t.Error("not valid url ", arr[i])
			}
		}

	}
}
