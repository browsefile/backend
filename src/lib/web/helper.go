package web

import (
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/utils"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
)

func SetupHandler(cfg *config.GlobalConfig) http.Handler {
	fb := &lib.FileBrowser{
		Config:    cfg,
		ReCaptcha: &lib.ReCaptcha{cfg.CaptchaConfig.Host, cfg.CaptchaConfig.Key, cfg.CaptchaConfig.Secret},
		NewFS: func(scope string) lib.FileSystem {
			return utils.Dir(scope)
		},
	}
	DavHandler(fb)
	needUpd, err := fb.Setup()
	if err != nil {
		log.Fatal(err)
	}
	if needUpd {
		cfg.WriteConfig()
	}

	return Handler(fb)
}
func DavHandler(fb *lib.FileBrowser) {
	ramLock := webdav.NewMemLS()
	for _, u := range fb.Config.Users {
		u.DavHandler = &webdav.Handler{
			FileSystem: webdav.Dir(fb.Config.GetDavPath(u.Username)),
			LockSystem: ramLock,
			Logger:     config.DavLogger,
		}
	}
}
