package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/browsefile/backend/src/cnst"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	updateLock = new(sync.RWMutex)
)
var config *GlobalConfig
var usersRam map[string]*UserConfig
var DavLogger func(r *http.Request, err error)

/*
Single config for everything.
update automatically
*/
type GlobalConfig struct {
	Users   []*UserConfig `json:"users"`
	Http    *ListenConf   `json:"http"`
	Tls     *ListenConf   `json:"https"`
	Log     string        `json:"log"`
	TLSKey  string        `json:"tlsKey"`
	TLSCert string        `json:"tlsCert"`
	// Scope is the Path the user has access to.
	FilesPath      string `json:"filesPath"`
	*CaptchaConfig `json:"captchaConfig"`
	*Auth          `json:"auth"`
	*PreviewConf   `json:"preview"`
	//http://host:port that used behind DMZ
	ExternalShareHost string `json:"externalShareHost"`

	//Path to config file
	Path string `json:"-"`
}

type ListenConf struct {
	Port int    `json:"port"`
	IP   string `json:"ip"`
	// Define if which of the following authentication mechansims should be used:
	// - 'default', which requires a user and a password.
	// - 'proxy', which requires a valid user and the user name has to be provided through an
	//   web header.
	// - 'none', which allows anyone to access the filebrowser instance.
	// If 'Method' is set to 'proxy' the header configured below is used to identify the user.
	AuthMethod string `json:"authMethod"`
}

func (l *ListenConf) copy() *ListenConf {
	return &ListenConf{l.Port, l.IP, l.AuthMethod}

}

// Auth settings.
type PreviewConf struct {
	//enable preview generating by call .sh
	ScriptPath string `json:"scriptPath"`
	Threads    int    `json:"threads"`
	FirstRun   bool   `json:"previewOnFirstRun"`
}

// Auth settings.
type Auth struct {
	Header string `json:"header"`
	Key    string `json:"key"`
}

// ~/<<cfg_PATH>>/<<username>>/
func (cfg *GlobalConfig) GetDavPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName)

}

//since we sure that this method will not modify, just return original
func (cfg *GlobalConfig) GetExternal(hash string) (res *ShareItem, usr *UserConfig) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	for _, user := range cfg.Users {
		for _, item := range user.Shares {
			if hash == item.Hash {
				res = item
				usr = user
				break
			}
		}
		if res != nil {
			break
		}
	}

	return res, usr

}

func (auth *Auth) copyAuth() *Auth {
	return &Auth{
		Key:    auth.Key,
		Header: auth.Header,
	}

}
func (c *CaptchaConfig) copyCaptchaConfig() *CaptchaConfig {
	return &CaptchaConfig{
		Key:    c.Key,
		Secret: c.Secret,
		Host:   c.Host,
	}

}

type CaptchaConfig struct {
	Host   string `json:"host"`
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

func (cfg *GlobalConfig) Verify() {
	updateLock.RLock()
	defer updateLock.RUnlock()
	for _, u := range cfg.Users {
		for _, shr := range u.Shares {
			shr.Path = strings.TrimSuffix(shr.Path, "/")
			shr.Hash = GenShareHash(u.Username, shr.Path)
		}
	}
}

// ~/<<cfg_PATH>>/<<username>>/files
func (cfg *GlobalConfig) GetUserHomePath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "files")
}

// ~/<<cfg_PATH>>/<<username>>/shares
func (cfg *GlobalConfig) GetUserSharesPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "shares")
}

// ~/<<cfg_PATH>>/<<username>>/preview
func (cfg *GlobalConfig) GetUserPreviewPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "preview")
}

// ~/<<cfg_PATH>>/<<username>>/sharex
func (cfg *GlobalConfig) GetUserSharexPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "sharex")
}

//read and initiate global config, if file missed, one will be created with default settings.
func (cfg *GlobalConfig) ReadConfigFile() {
	var paths []string
	argumentPath := len(cfg.Path) > 0
	if len(cfg.Path) > 0 {
		paths = append(paths, cfg.Path)
		argumentPath = true
	}
	paths = append(paths, cnst.FilePath1, cnst.FilePath2)
	if !argumentPath {
		curPath, err := os.Getwd()
		if err == nil {
			paths = append(paths, filepath.Join(curPath, cnst.FilePath1))
		}
	}
	for i, p := range paths {
		err := cfg.parseConf(p)
		if err != nil && i < len(paths)-1 {
			continue
		}
		cfg.Path = p
		//if no paths fit, then try to use current process folder or path that set from cmd argument(preferred)
		if i == len(paths)-1 && err != nil {
			if argumentPath {
				cfg.Path = paths[0]
			}
			cfg.FilesPath = filepath.Join(filepath.Dir(cfg.Path), "bf-data")

			// DefaultUser is used on New, when no 'config' exists
			cfg.Users = append(cfg.Users, &UserConfig{
				Username:  "admin",
				AllowNew:  true,
				Admin:     true,
				AllowEdit: true,
				FirstRun:  true,
				Password:  "admin",
				ViewMode:  "mosaic",
				Locale:    "en",
			})
			cfg.Http = &ListenConf{AuthMethod: "default", IP: "127.0.0.1", Port: 8999}
			cfg.Tls = &ListenConf{AuthMethod: "", IP: "", Port: 0}
			cfg.ExternalShareHost = "http://127.0.0.1:8999"
			cfg.PreviewConf = &PreviewConf{Threads: 2, ScriptPath: filepath.Join(filepath.Dir(cfg.Path), "bfconvert.sh")}
			cfg.CaptchaConfig = &CaptchaConfig{}
			cfg.Auth = &Auth{Header: "X-Forwarded-User"}
			cfg.Log = "stdout"
		}
		break
	}
	fmt.Println("using config at path : " + cfg.Path)

	config = cfg
	cfg.RefreshUserRam()
	cfg.setupLog()
	cfg.Verify()
	cfg.setUpPaths()

}

//setup paths for all users, validate shares, and symlinks
func (cfg *GlobalConfig) setUpPaths() {
	needUpdate := false
	for _, u := range cfg.Users {
		//rebuild share paths
		p := cfg.GetUserSharesPath(u.Username)
		_ = os.RemoveAll(p)
	}
	for _, u := range cfg.Users {
		//create shares folder
		createPath(cfg.GetUserSharesPath(u.Username))
		//create external shares folder
		createPath(cfg.GetUserSharexPath(u.Username))
		//create user files folder
		createPath(cfg.GetUserHomePath(u.Username))
		//create user preview folder
		createPath(cfg.GetUserPreviewPath(u.Username))

		_ = cfg.checkDavFolder(u);
		//fix bad symlinks, or build missed for share for specific user
		for _, owner := range cfg.Users {
			//skip same user
			if owner.Username == u.Username {
				continue
			}

			for _, shr := range u.Shares {
				if !cfg.checkShareSymLinkPath(shr, owner.Username, u.Username) {
					u.DeleteShare(shr.Path)

					needUpdate = true
				}
			}
		}
		for _, shr := range u.Shares {
			if !cfg.checkExternalShareSymLinkPath(shr, u.Username) {
				u.DeleteShare(shr.Path)
				needUpdate = true
			}
		}
	}
	if needUpdate {
		cfg.WriteConfig()
	}
}
func createPath(p string) (ok bool) {
	ok = true
	if err := os.MkdirAll(p, cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
		log.Println("config: ", err)
		ok = false
	}

	return ok
}

//create symlinks at dav path for shares, and files folders from user's filesystem
func (cfg *GlobalConfig) checkDavFolder(u *UserConfig) (err error) {
	dp := cfg.GetDavPath(u.Username)
	sharePath := filepath.Join(dp, cnst.WEB_DAV_FOLDER, "shares")
	filesPath := filepath.Join(dp, cnst.WEB_DAV_FOLDER, "files")

	//check symlink from user home dir, to the user's webdav Path
	createPath(filesPath)
	//check shares symlink path
	createPath(sharePath)

	if _, err = os.Readlink(filesPath); err != nil {
		_ = os.Remove(filesPath)
	}
	if err = os.Symlink(cfg.GetUserHomePath(u.Username), filesPath); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create user symlink at dav ", err)
	}
	//recreate shares path
	_ = os.Remove(sharePath)
	if err = os.Symlink(cfg.GetUserSharesPath(u.Username), sharePath); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create user symlink at dav ", err)
	}
	return
}

func (cfg *GlobalConfig) parseConf(p string) (r error) {
	if jsonFile, err := os.Open(p); err == nil {
		byteValue, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(byteValue, &cfg)
		r = jsonFile.Close()
	} else {
		fmt.Printf("can't open %s", p)
		fmt.Println(err)
		r = err
	}

	return r
}

func (cfg *GlobalConfig) GetAdmin() *UserConfig {
	for _, usr := range cfg.Users {
		if usr.Admin {
			return usr
		}
	}
	return nil
}

func (cfg *GlobalConfig) WriteConfig() {
	updateLock.Lock()
	defer updateLock.Unlock()
	//todo check hash if config changed
	jsonData, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Println(err)
	} else {
		err = os.MkdirAll(filepath.Dir(cfg.Path), cnst.PERM_DEFAULT)
		if err != nil {
			log.Println(err)
		}
		err = ioutil.WriteFile(cfg.Path, jsonData, cnst.PERM_DEFAULT)
		if err != nil {
			log.Println("config : cant write config file", err)
		}
	}
}

//should not be called directly
func (cfg *GlobalConfig) RefreshUserRam() {
	usersRam = make(map[string]*UserConfig)
	for _, u := range cfg.Users {
		//index usernames
		usersRam[u.Username] = u
		for _, ip := range u.IpAuth {
			//index ips as well
			usersRam[ip] = u
		}

	}
}
func (cfg *GlobalConfig) setupLog() {
	// Set up process log before anything bad happens.
	switch cfg.Log {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	case "":
		log.SetOutput(ioutil.Discard)
	default:
		log.SetOutput(&lumberjack.Logger{
			LocalTime:  true,
			Filename:   cfg.Log,
			MaxSize:    100,
			MaxAge:     14,
			MaxBackups: 10,
		})
	}
	DavLogger = func(r *http.Request, err error) {
		if err != nil {
			log.Printf("WEBDAV req: %v\n\nERROR: %v", r, err)
			log.Printf(r.URL.Path)
		}
	}
}

//returns salt key in bytes, err in case key missed
func (cfg *GlobalConfig) GetKeyBytes() ([]byte, error) {
	updateLock.RLock()
	defer updateLock.RUnlock()
	if len(cfg.Auth.Key) == 0 {
		return nil, cnst.ErrEmptyKey
	}
	return base64.StdEncoding.DecodeString(cfg.Auth.Key)
}

//clone config
func (cfg *GlobalConfig) CopyConfig() *GlobalConfig {
	updateLock.RLock()
	defer updateLock.RUnlock()
	res := &GlobalConfig{
		Users:             cfg.GetUsers(),
		Http:              &ListenConf{cfg.Http.Port, cfg.Http.IP, cfg.Http.AuthMethod},
		Log:               cfg.Log,
		CaptchaConfig:     cfg.copyCaptchaConfig(),
		Auth:              cfg.copyAuth(),
		PreviewConf:       &PreviewConf{ScriptPath: cfg.ScriptPath, Threads: cfg.Threads},
		FilesPath:         cfg.FilesPath,
		TLSKey:            cfg.TLSKey,
		TLSCert:           cfg.TLSCert,
		ExternalShareHost: cfg.ExternalShareHost,
		Path:              cfg.Path,
	}
	if cfg.Tls != nil {
		res.Tls = &ListenConf{cfg.Tls.Port, cfg.Tls.IP, cfg.Tls.AuthMethod}
	} else {
		res.Tls = &ListenConf{0, "", ""}
	}

	return res
}

//deep update config
func (cfg *GlobalConfig) UpdateConfig(u *GlobalConfig) {
	updateLock.Lock()
	defer updateLock.Unlock()
	cfg.Http = u.Http.copy()
	cfg.Tls = u.Tls.copy()
	cfg.Log = u.Log
	cfg.Auth = u.copyAuth()
	cfg.CaptchaConfig = u.copyCaptchaConfig()
	cfg.FilesPath = u.FilesPath
	cfg.TLSCert = u.TLSCert
	cfg.TLSKey = u.TLSKey
	cfg.PreviewConf = u.PreviewConf
	cfg.ExternalShareHost = u.ExternalShareHost
}

//update salt key
func (cfg *GlobalConfig) SetKey(k []byte) {
	updateLock.Lock()
	defer updateLock.Unlock()
	cfg.Auth.Key = base64.StdEncoding.EncodeToString(k)
}
