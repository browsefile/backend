package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/browsefile/backend/src/cnst"
	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	ExternalShareHost string        `json:"externalShareHost"`
	updateLock        *sync.RWMutex `json:"-"`
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

func (gc *GlobalConfig) GetDavPath(userName string) string {
	return filepath.Join(gc.FilesPath, userName)

}
func (gc *GlobalConfig) DeleteShare(usr *UserConfig, p string) (res bool) {
	gc.lock()
	defer gc.unlock()
	res = usr.deleteShare(p)
	if res {
		gc.RefreshUserRam()
	}

	return
}

//since we sure that this method will not modify, just return original
func (gc *GlobalConfig) GetExternal(hash string) (res *ShareItem, usr *UserConfig) {
	gc.lockR()
	defer gc.unlockR()
	for _, user := range gc.Users {
		for _, item := range user.Shares {
			if strings.EqualFold(hash, item.Hash) {
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

func (auth *Auth) CopyAuth() *Auth {
	return &Auth{
		Key:    auth.Key,
		Header: auth.Header,
	}

}
func (c *CaptchaConfig) CopyCaptchaConfig() *CaptchaConfig {
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
	for _, u := range cfg.Users {
		for _, shr := range u.Shares {
			shr.Path = strings.TrimSuffix(shr.Path, "/")
		}
	}

	//todo
}
func (cfg *GlobalConfig) GetUserHomePath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "files")
}
func (cfg *GlobalConfig) GetUserSharesPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "shares")
}
func (cfg *GlobalConfig) GetUserPreviewPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "preview")
}
func (cfg *GlobalConfig) GetSharePreviewPath(url string) (res string) {
	//cut username
	u := strings.TrimPrefix(url, "/")
	if len(u) > 0 {
		arr := strings.Split(u, "/")
		if len(arr) > 1 {
			user, ok := cfg.GetByUsername(arr[0])
			if ok {
				shrPath := strings.Replace(u, arr[0], "", 1)
				res = filepath.Join(cfg.GetUserPreviewPath(user.Username), shrPath)
			}
		}
	}

	return res
}

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

	cfg.updateLock = new(sync.RWMutex)
	config = cfg
	cfg.RefreshUserRam()
	cfg.setupLog()
	cfg.setUpPaths()

}
func (cfg *GlobalConfig) setUpPaths() {
	needUpdate := false
	for _, u := range cfg.Users {
		//rebuild shares paths
		p := cfg.GetUserSharesPath(u.Username)
		os.RemoveAll(p)
	}
	for _, u := range cfg.Users {
		//create shares folder
		if err := os.MkdirAll(cfg.GetUserSharesPath(u.Username), cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
			log.Println("config : Cant create share path for user "+u.Username, err)
		}
		//create user files folder
		if err := os.MkdirAll(cfg.GetUserHomePath(u.Username), cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
			log.Println("config : Cant create files path for user "+u.Username, err)
		}

		cfg.checkDavFolder(u);
		//fix bad symlinks, or build missed for share for specific user
		for _, owner := range cfg.Users {
			//skip same user
			if strings.EqualFold(owner.Username, u.Username) {
				continue
			}

			for _, shr := range u.Shares {
				if !cfg.checkShareSymLinkPath(shr, owner.Username, u.Username) {
					u.deleteShare(shr.Path)

					needUpdate = true
				}
			}
		}
	}
	if needUpdate {
		cfg.WriteConfig()
	}
}
func (cfg *GlobalConfig) checkDavFolder(u *UserConfig) (err error) {
	dp := cfg.GetDavPath(u.Username)
	sharePath := filepath.Join(dp, cnst.WEB_DAV_FOLDER, "shares")
	oFlsPath := filepath.Join(dp, cnst.WEB_DAV_FOLDER, "files")

	if err = os.MkdirAll(oFlsPath, cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create files path, at dav ", err)
		//check symlink from user home dir, to the user's webdav Path
	}
	if _, err = os.Readlink(oFlsPath); err != nil {
		os.Remove(oFlsPath)
	}
	if err = os.Symlink(cfg.GetUserHomePath(u.Username), oFlsPath); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create user files path, at dav ", err)
	}

	//check shares symlink
	if err = os.MkdirAll(sharePath, cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create shares path, at dav ", err)
	}

	if _, err = os.Readlink(sharePath); err != nil {
		os.Remove(sharePath)
	}
	if err = os.Symlink(cfg.GetUserSharesPath(u.Username), sharePath); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create user files path, at dav ", err)
	}
	return
}

//returns true in case share good, otherwise original share path does not exists
func (cfg *GlobalConfig) checkShareSymLinkPath(shr *ShareItem, shrUser, owner string) (res bool) {
	res = true
	if strings.EqualFold(owner, shrUser) {
		return
	}
	var err error
	dp := filepath.Join(cfg.GetUserSharesPath(shrUser), owner)
	//check basic folder exists
	if err = os.MkdirAll(dp, cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
		log.Printf("config : Cant create share path at %s ", dp)

	} else {
		//destination path for symlink
		dPath := filepath.Join(dp, shr.Path)
		//source path for symlink
		sPath := filepath.Join(cfg.GetUserHomePath(owner), shr.Path)
		_ = os.RemoveAll(dPath)
		//take the parent folder
		_ = os.MkdirAll(filepath.Dir(dPath), cnst.PERM_DEFAULT)
		if shr.IsActive() && shr.IsAllowed(shrUser) {
			//check if share valid
			if _, err = os.Stat(sPath); err != nil && os.IsNotExist(err) {
				res = false
			} else if err = os.Symlink(sPath, dPath); err != nil && !os.IsExist(err) {
				log.Printf("config : Cant create share sym link from '%s' TO '%s'", sPath, dPath)
			}
		}
	}
	return res
}

func (cfg *GlobalConfig) parseConf(p string) (r error) {
	if jsonFile, err := os.Open(p); err == nil {
		byteValue, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(byteValue, &cfg)
		jsonFile.Close()
	} else {
		fmt.Printf("can't open %s", p)
		fmt.Println(err)
		r = err
	}

	return r
}
func (cfg *GlobalConfig) Init() {
	config = cfg
	for _, u := range cfg.Users {
		for _, shr := range u.Shares {
			shr.Hash = GenShareHash(u.Username, shr.Path)
		}
	}
	cfg.updateLock = new(sync.RWMutex)
	cfg.RefreshUserRam()

}
func (cfg *GlobalConfig) GetAdmin() *UserConfig {
	cfg.lockR()
	defer cfg.unlockR()
	for _, usr := range cfg.Users {
		if usr.Admin {
			return usr
		}
	}
	return nil
}

func (cfg *GlobalConfig) WriteConfig() {
	cfg.lock()
	defer cfg.unlock()
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
			Filename:   cfg.Log,
			MaxSize:    100,
			MaxAge:     14,
			MaxBackups: 10,
		})
	}
	DavLogger = func(r *http.Request, err error) {
		if err != nil {
			log.Printf("WEBDAV: %#s, ERROR: %v", r, err)
			log.Printf(r.URL.Path)
		}
	}
}
func (cfg *GlobalConfig) GetByUsername(username string) (*UserConfig, bool) {
	cfg.lockR()
	defer cfg.unlockR()

	if username == cnst.GUEST {
		admin := cfg.GetAdmin()
		return &UserConfig{
			Username:  username,
			Locale:    admin.Locale,
			Admin:     false,
			ViewMode:  admin.ViewMode,
			AllowNew:  false,
			AllowEdit: false,
		}, true
	}

	res, ok := usersRam[username]
	if !ok {
		return nil, ok
	}

	return res.copyUser(), ok
}
func (cfg *GlobalConfig) GetByIp(ip string) (*UserConfig, bool) {
	cfg.lockR()
	defer cfg.unlockR()
	ip = strings.Split(ip, ":")[0]
	res, ok := usersRam[ip]
	if !ok {
		return nil, ok
	}

	return res.copyUser(), ok
}

func (cfg *GlobalConfig) GetUsers() (res []*UserConfig) {
	cfg.lockR()
	defer cfg.unlockR()
	res = make([]*UserConfig, len(cfg.Users))
	for i, u := range cfg.Users {
		res[i] = u.copyUser()
	}

	return res
}

func (cfg *GlobalConfig) Add(u *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()
	_, exists := usersRam[u.Username]
	if exists {
		return errors.New("User exists " + u.Username)
	}

	cfg.Users = append(cfg.Users, u)
	cfg.RefreshUserRam()

	return nil
}
func (cfg *GlobalConfig) UpdatePassword(u *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()
	i := cfg.getUserIndex(u.Username)
	if i >= 0 {
		//update only specific fields
		cfg.Users[i].Password = u.Password
	} else {
		return errors.New("User does not exists " + u.Username)
	}
	return nil
}
func (cfg *GlobalConfig) Update(u *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()

	i := cfg.getUserIndex(u.Username)
	if i >= 0 {
		//update only specific fields
		cfg.Users[i].Admin = u.Admin
		cfg.Users[i].ViewMode = u.ViewMode
		cfg.Users[i].FirstRun = u.FirstRun
		cfg.Users[i].Shares = u.Shares
		cfg.Users[i].IpAuth = u.IpAuth
		cfg.Users[i].Locale = u.Locale
		cfg.Users[i].AllowEdit = u.AllowEdit
		cfg.Users[i].AllowNew = u.AllowNew
		cfg.Users[i].LockPassword = u.LockPassword
		cfg.Users[i].UID = u.UID
		cfg.Users[i].GID = u.GID
		cfg.RefreshUserRam()
	} else {
		return errors.New("User does not exists " + u.Username)
	}

	return nil
}

func (cfg *GlobalConfig) UpdateUsers(users []*UserConfig) error {
	cfg.lock()
	defer cfg.unlock()
	if len(users) > 0 {
		cfg.Users = users
	}

	cfg.RefreshUserRam()
	return nil
}

func (cfg *GlobalConfig) Delete(username string) error {
	cfg.lock()
	defer cfg.unlock()

	i := cfg.getUserIndex(username)
	if i >= 0 {
		cfg.Users = append(cfg.Users[:i], cfg.Users[i+1:]...)
	}
	cfg.RefreshUserRam()

	return nil
}
func (cfg *GlobalConfig) getUserIndex(userName string) int {
	for i, u := range cfg.Users {
		if strings.EqualFold(u.Username, userName) {
			return i
		}
	}
	return -1
}

func (cfg *GlobalConfig) GetKeyBytes() ([]byte, error) {
	if len(cfg.Auth.Key) == 0 {
		return nil, cnst.ErrEmptyKey
	}
	return base64.StdEncoding.DecodeString(cfg.Auth.Key)
}

func (cfg *GlobalConfig) CopyConfig() *GlobalConfig {
	cfg.lockR()
	defer cfg.unlockR()

	res := &GlobalConfig{
		Users:             cfg.GetUsers(),
		Http:              &ListenConf{cfg.Http.Port, cfg.Http.IP, cfg.Http.AuthMethod},
		Log:               cfg.Log,
		CaptchaConfig:     cfg.CopyCaptchaConfig(),
		Auth:              cfg.CopyAuth(),
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
func (cfg *GlobalConfig) UpdateConfig(u *GlobalConfig) {
	cfg.lock()
	defer cfg.unlock()
	cfg.Http = u.Http.copy()
	cfg.Tls = u.Tls.copy()
	cfg.Log = u.Log
	cfg.Auth = u.CopyAuth()
	cfg.CaptchaConfig = u.CopyCaptchaConfig()
	cfg.FilesPath = u.FilesPath
	cfg.TLSCert = u.TLSCert
	cfg.TLSKey = u.TLSKey
	cfg.PreviewConf = u.PreviewConf
	cfg.ExternalShareHost = u.ExternalShareHost
}

func (cfg *GlobalConfig) SetKey(k []byte) {
	cfg.Auth.Key = base64.StdEncoding.EncodeToString(k)
}
func (cfg *GlobalConfig) lock() {
	cfg.updateLock.Lock()
}
func (cfg *GlobalConfig) unlock() {
	//cfg.WriteConfig()
	cfg.updateLock.Unlock()
}

func (cfg *GlobalConfig) lockR() {
	cfg.updateLock.RLock()
}
func (cfg *GlobalConfig) unlockR() {
	cfg.updateLock.RUnlock()
}
