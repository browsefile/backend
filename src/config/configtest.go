package config

import (
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib/utils"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//helper struct for testing only
type TContext struct {
	SharePathDeep  string
	SharePathUp    string
	ConfigPath     string
	Usr1           *UserConfig
	Usr2           *UserConfig
	Guest          *UserConfig
	User1FS        utils.Dir
	User1FSPreview utils.Dir
	User2FS        utils.Dir
	User2FSShare   utils.Dir
	AdminFSShare   utils.Dir
	AdminFS        utils.Dir
	*GlobalConfig
	Srv   *httptest.Server
	Tr    *http.Transport
	Token string
}

func (tc *TContext) Init() {
	tc.SharePathUp = "/test"
	tc.SharePathDeep = "/test/share"
	tc.ConfigPath, _ = ioutil.TempDir("", "bf_")
	cfg := GlobalConfig{
		Path:      tc.ConfigPath + "/bf_test.json",
		FilesPath: tc.ConfigPath,
	}
	cfg.ReadConfigFile()
	cfg.Auth = &Auth{Key: "ae12354"}

	tc.GlobalConfig = &cfg
}
func (tc *TContext) Clean(t *testing.T) {
	err := os.RemoveAll(tc.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if tc.Srv != nil {
		tc.Srv.Close()
	}
}

//setup filesystem in ram for users, will create empty test files
func (tc *TContext) InitWithUsers(t *testing.T) {
	tc.Init()

	previewSH, err := os.Getwd()
	if err != nil {
		t.Log(err)
	}
	previewSH = strings.TrimSuffix(previewSH, "/src/lib/web")
	t.Log(previewSH)
	tc.PreviewConf.ScriptPath = previewSH + "/bfconvert.sh"

	tc.Usr1 = tc.MakeUser("user1")
	tc.Usr1.AllowNew = true
	tc.Usr1.AllowEdit = true
	tc.Usr2 = tc.MakeUser("user2")
	//create users
	_ = tc.AddUser(tc.Usr1)
	_ = tc.AddUser(tc.Usr2)
	tc.User1FS = utils.Dir(tc.GetUserHomePath(tc.Usr1.Username))
	tc.User1FSPreview = utils.Dir(tc.GetUserPreviewPath(tc.Usr1.Username))
	tc.User2FS = utils.Dir(tc.GetUserHomePath(tc.Usr2.Username))
	tc.AdminFS = utils.Dir(tc.GetUserHomePath(tc.GetAdmin().Username))
	tc.User2FSShare = utils.Dir(tc.GetUserSharesPath(tc.Usr2.Username))
	tc.AdminFSShare = utils.Dir(tc.GetUserSharesPath(tc.GetAdmin().Username))
	tc.Guest, _ = tc.GetUserByUsername("guest")
	//create paths for share item for 2 users
	err = tc.User1FS.Mkdir(tc.SharePathDeep, cnst.PERM_DEFAULT, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = tc.User2FS.Mkdir(tc.SharePathDeep, cnst.PERM_DEFAULT, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = tc.AdminFS.Mkdir(tc.SharePathDeep, cnst.PERM_DEFAULT, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	//create some real image for preview gen test
	pixels := make([]byte, 500*500) // slice of your gray pixels, size of 100x100v
	_, err = rand.Read(pixels)
	if err != nil {
		t.Log(err)
	}
	img := image.NewGray(image.Rect(0, 0, 500, 500))
	img.Pix = pixels
	//share2 to admin user
	f, _ := tc.User1FS.OpenFile(tc.SharePathDeep+"/real.jpg", os.O_WRONLY|os.O_CREATE, cnst.PERM_DEFAULT, 0, 0)
	if err != nil {
		t.Log(err)
	}
	err = jpeg.Encode(f, img, nil)
	if err != nil {
		t.Log(err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			t.Log(err)
		}
	}()
	//create empty files
	for _, f := range []string{"t.jpg", "t.mp3", "t.png", "t.mp4", "t.txt", "t.pdf", "t.doc"} {
		for _, p := range []string{tc.SharePathUp, tc.SharePathDeep, "/"} {
			p = filepath.Join(p, f)
			inf, _ := tc.User1FS.OpenFile(p, os.O_RDONLY|os.O_CREATE, cnst.PERM_DEFAULT, 0, 0)
			_ = inf.Close()
			inf, _ = tc.User2FS.OpenFile(p, os.O_RDONLY|os.O_CREATE, cnst.PERM_DEFAULT, 0, 0)
			_ = inf.Close()
			inf, _ = tc.AdminFS.OpenFile(p, os.O_RDONLY|os.O_CREATE, cnst.PERM_DEFAULT, 0, 0)
			_ = inf.Close()

		}

	}

}
func (tc *TContext) MakeUser(name string) *UserConfig {
	return &UserConfig{Username: name, Password: "1", FirstRun: true, Locale: "en", ViewMode: "mosaic"}
}
