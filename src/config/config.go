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
	Port       int    `json:"port"`
	IP         string `json:"ip"`
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
	// Define if which of the following authentication mechansims should be used:
	// - 'default', which requires a user and a password.
	// - 'proxy', which requires a valid user and the user name has to be provided through an
	//   web header.
	// - 'none', which allows anyone to access the filebrowser instance.
	// If 'Method' is set to 'proxy' the header configured below is used to identify the user.
	Header string `json:"header"`

	Key string `json:"key"`
}

func (gc *GlobalConfig) GetDavPath(userName string) string {
	return filepath.Join(gc.FilesPath, userName)

}
func (gc *GlobalConfig) DeleteShare(usr *UserConfig, p string) (res bool) {
	config.lock()
	defer config.unlock()
	res = usr.deleteShare(p)
	if res {
		usr.sortShares()
		gc.RefreshUserRam()
	}

	return
}

//since we sure that this method will not modify, just return original
func (gc *GlobalConfig) GetExternal(hash string) (res *ShareItem, usr *UserConfig) {
	config.lockR()
	defer config.unlockR()
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

	if len(cfg.Path) > 0 {
		paths = append(paths, cfg.Path)
	}
	paths = append(paths, cnst.FilePath1, cnst.FilePath2)
	for _, p := range paths {
		jsonFile, err := os.Open(p)
		defer jsonFile.Close()
		if err != nil {
			fmt.Println("can't open " + p)
			fmt.Println(err)
		} else {
			err = cfg.parseConf(jsonFile)
			if err != nil {
				fmt.Print("failed to parse config at " + p)
				continue
			}
			cfg.Path = p
			break
		}
	}
	fmt.Println("using config at path : " + cfg.Path)
	cfg.RefreshUserRam()
	cfg.updateLock = new(sync.RWMutex)
	config = cfg

	for _, u := range cfg.Users {
		for _, shr := range u.Shares {
			shr.Path = strings.TrimSuffix(shr.Path, "/")
		}
		u.sortShares()
	}
	cfg.setupLog()
	cfg.setUpPaths()

}
func (cfg *GlobalConfig) setUpPaths() {
	for _, u := range config.Users {
		//create shares folder
		if err := os.MkdirAll(cfg.GetUserSharesPath(u.Username), cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
			log.Println("config : Cant create share path for user ", err)
		}

		cfg.checkDavFolder(u);
		//fix bad symlinks, or build missed for share for specific user
		for _, owner := range config.Users {
			//skip same user
			if strings.EqualFold(owner.Username, u.Username) {
				continue
			}

			for _, shr := range u.Shares {
				cfg.checkShareSymLinkPath(shr, owner.Username, u.Username)
			}
		}
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

func (cfg *GlobalConfig) checkShareSymLinkPath(shr *ShareItem, user, owner string) {
	if strings.EqualFold(owner, user) {
		return
	}
	var err error
	dp := filepath.Join(cfg.GetUserSharesPath(user), owner)
	//check basic folder exists
	if err = os.MkdirAll(dp, cnst.PERM_DEFAULT); err != nil && !os.IsExist(err) {
		log.Println("config : Cant create share path for userF ", err)
	} else {
		//destination path for symlink
		dPath := filepath.Join(dp, shr.Path)
		//source path for symlink
		sPath := filepath.Join(cfg.GetUserHomePath(owner), shr.Path)
		_ = os.Remove(dPath)
		//take the parent folder
		_ = os.MkdirAll(filepath.Dir(dPath), cnst.PERM_DEFAULT)
		if shr.IsActive() && shr.IsAllowed(user) {
			if err = os.Symlink(sPath, dPath); err != nil && !os.IsExist(err) {
				log.Println("config : Cant create share sym link at ", err)
			}

		}
	}
}

func (cfg *GlobalConfig) parseConf(jsonFile *os.File) (err error) {
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		fmt.Print("can't parse " + cfg.Path)
		fmt.Print(err)
	}
	return
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
		admin := config.GetAdmin()
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
	return &GlobalConfig{
		Users:             cfg.GetUsers(),
		Http:              &ListenConf{cfg.Http.Port, cfg.Http.IP, cfg.Http.AuthMethod},
		Tls:               &ListenConf{cfg.Tls.Port, cfg.Tls.IP, cfg.Tls.AuthMethod},
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
