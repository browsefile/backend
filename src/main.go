package main

import (
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/browsefile/backend/src/lib/web"
	"log"
	"net"
	"net/http"
	"strconv"
)

func main() {
	cfg := new(config.GlobalConfig)
	cfg.ReadConfigFile(config.FileName)
	cfg.SetupLog()
	cfg.Verify()
	// Builds the address and a listener.
	listener, err := net.Listen("tcp", cfg.IP+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		log.Fatal(err)
	}

	// Tell the user the port in which is listening.
	log.Println("Listening on", listener.Addr().String())

	if len(cfg.TLSCert) > 0 && len(cfg.TLSCert) > 0 {
		err = http.ServeTLS(listener, handler(cfg), cfg.TLSCert, cfg.TLSKey)
	} else {
		err = http.Serve(listener, handler(cfg))
	}
	// Starts the server.
	if err != nil {
		log.Fatal(err)
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

	needUpd, err := fb.Setup()
	if err != nil {
		log.Fatal(err)
	}
	if needUpd {
		fb.Config.Store()
	}

	cfg.StartMonitor()

	return web.Handler(fb)
}
