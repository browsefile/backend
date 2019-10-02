package main

import (
	"fmt"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/browsefile/backend/src/lib/web"
	"golang.org/x/net/webdav"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	cfg := new(config.GlobalConfig)
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" {
			fmt.Printf("Default config file locations : '%s', '%s'. Also you can specify own by passing path as first argument.", cnst.FilePath1, cnst.FilePath2)
			os.Exit(0)
		} else {
			cfg.Path = os.Args[1]
		}
	}

	cfg.ReadConfigFile()
	cfg.Verify()
	var listener, listenerTLS net.Listener
	var err error
	isHttp := cfg.Http != nil && cfg.Http.Port > 0
	isTLS := cfg.Tls != nil && cfg.Tls.Port > 0 && len(cfg.TLSCert) > 0 && len(cfg.TLSCert) > 0
	// Builds the address and a listener.
	if isHttp {
		listener, err = net.Listen("tcp", cfg.Http.IP+":"+strconv.Itoa(cfg.Http.Port))
		if err != nil {
			log.Fatal(err)
		}
	}
	if isTLS {
		listenerTLS, err = net.Listen("tcp", cfg.Tls.IP+":"+strconv.Itoa(cfg.Tls.Port))
		if err != nil {
			log.Fatal(err)
		}
	}

	srv := &http.Server{Handler: handler(cfg), ReadTimeout: 5 * time.Hour, WriteTimeout: 5 * time.Hour}
	// Tell the user the port in which is listening.
	if isHttp {
		log.Println("Listening http://" + listener.Addr().String())
		log.Println("dav://" + listener.Addr().String() + cnst.WEB_DAV_URL)
	}
	if isTLS {
		log.Println()
		log.Println("Listening https://" + listenerTLS.Addr().String())
		log.Println("davs://" + listenerTLS.Addr().String() + cnst.WEB_DAV_URL)
		if isHttp {
			go func() {
				err = srv.ServeTLS(listenerTLS, cfg.TLSCert, cfg.TLSKey)
				if err != nil {
					log.Fatal(err)
				}
			}()
		} else {
			err = srv.ServeTLS(listenerTLS, cfg.TLSCert, cfg.TLSKey)
			if err != nil {
				log.Fatal(err)
			}
		}

	}
	if isHttp {
		err = srv.Serve(listener)

		// Starts the server.
		if err != nil {
			log.Fatal(err)
		}
	}

}

func handler(cfg *config.GlobalConfig) http.Handler {
	fb := &lib.FileBrowser{
		Config:    cfg,
		ReCaptcha: &lib.ReCaptcha{cfg.CaptchaConfig.Host, cfg.CaptchaConfig.Key, cfg.CaptchaConfig.Secret},
		NewFS: func(scope string) lib.FileSystem {
			return fileutils.Dir(scope)
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

	return web.Handler(fb)
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
