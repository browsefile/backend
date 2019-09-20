package config

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var config *GlobalConfig
var usersRam map[string]*UserConfig
var FileName = "browsefile.json"

const GUEST string = "guest"

/*
Single config for everything.
update automatically
*/
type GlobalConfig struct {
	Users   []*UserConfig `json:"users"`
	Port    int           `json:"port"`
	IP      string        `json:"ip"`
	Log     string        `json:"log"`
	TLSKey  string        `json:"tlsKey"`
	TLSCert string        `json:"tlsCert"`
	// Scope is the path the user has access to.
	FilesPath         string `json:"filesPath"`
	*CaptchaConfig    `json:"captchaConfig"`
	*Auth             `json:"auth"`
	*PreviewConf      `json:"preview"`
	ExternalShareHost string `json:"externalShareHost"`

	updateLock *sync.RWMutex `json:"-"`
}

// Auth settings.
type PreviewConf struct {
	//enable preview generating by call .sh
	AllowGeneratePreview bool `json:"allowGeneratePreview"`
	Threads              int  `json:"threads"`
	FirstRun             bool `json:"previewOnFirstRun"`
}

// Auth settings.
type Auth struct {
	// Define if which of the following authentication mechansims should be used:
	// - 'default', which requires a user and a password.
	// - 'proxy', which requires a valid user and the user name has to be provided through an
	//   web header.
	// - 'none', which allows anyone to access the filebrowser instance.
	Method string `json:"method"`
	// If 'Method' is set to 'proxy' the header configured below is used to identify the user.
	Header string `json:"header"`

	Key string `json:"key"`
}

func GenShareHash(userName, itmPath string) string {
	itmPath = strings.ReplaceAll(itmPath, "/", "")
	return base64.StdEncoding.EncodeToString(md5.New().Sum([]byte(userName + itmPath)))
}

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
		Method: auth.Method,
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
func (cfg *GlobalConfig) GetUserPreviewPath(userName string) string {
	return filepath.Join(cfg.FilesPath, userName, "preview")
}

func (cfg *GlobalConfig) ReadConfigFile(file string) {
	FileName = file
	// Open our jsonFile
	jsonFile, err := os.Open(file)
	defer jsonFile.Close()
	if err != nil {
		log.Print("can't open " + FileName)
		log.Print(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		log.Print("can't parse " + FileName)
		log.Print(err)
	}
	cfg.RefreshUserRam()
	cfg.updateLock = new(sync.RWMutex)
	config = cfg

	for _, u := range cfg.Users {
		u.sortShares()
	}
}
func (cfg *GlobalConfig) Init() {
	cfg.updateLock = new(sync.RWMutex)
	config = cfg
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
	//todo check hash if config changed
	jsonData, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		fmt.Println(err)
	} else {
		err = ioutil.WriteFile(FileName, jsonData, 0666)
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
func (cfg *GlobalConfig) SetupLog() {
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
}
func (cfg *GlobalConfig) GetByUsername(username string) (*UserConfig, bool) {
	cfg.lockR()
	defer cfg.unlockR()

	if username == GUEST {
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

	return res, ok
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
func (cfg *GlobalConfig) Update(u *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()
	_, exists := usersRam[u.Username]
	if !exists {
		return errors.New("User does not exists " + u.Username)
	}

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
	}

	cfg.RefreshUserRam()
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
	if strings.EqualFold(cfg.Users[i].Username, username) {
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
		return nil, errors.New("Key is empty")
	}
	return base64.StdEncoding.DecodeString(cfg.Auth.Key)
}

func (cfg *GlobalConfig) CopyConfig() *GlobalConfig {
	cfg.lockR()
	defer cfg.unlockR()
	return &GlobalConfig{
		Users:             cfg.GetUsers(),
		Port:              cfg.Port,
		IP:                cfg.IP,
		Log:               cfg.Log,
		CaptchaConfig:     cfg.CopyCaptchaConfig(),
		Auth:              cfg.CopyAuth(),
		PreviewConf:       &PreviewConf{AllowGeneratePreview: cfg.AllowGeneratePreview, Threads: cfg.Threads},
		FilesPath:         cfg.FilesPath,
		TLSKey:            cfg.TLSKey,
		TLSCert:           cfg.TLSCert,
		ExternalShareHost: cfg.ExternalShareHost,
	}
}
func (cfg *GlobalConfig) UpdateConfig(u *GlobalConfig) {
	cfg.lock()
	defer cfg.unlock()
	cfg.Port = u.Port
	cfg.IP = u.IP
	cfg.Log = u.Log
	cfg.Users = u.GetUsers()
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
	cfg.WriteConfig()
	cfg.updateLock.Unlock()

}

func (cfg *GlobalConfig) lockR() {
	cfg.updateLock.RLock()
}
func (cfg *GlobalConfig) unlockR() {
	cfg.updateLock.RUnlock()
}
