package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var config *GlobalConfig
var usersRam map[string]*UserConfig
var FileName = "browsefile.json"

/*
Single config for everything.
update automatically
*/
type GlobalConfig struct {
	Users          []*UserConfig `json:"users"`
	DefaultUser    *UserConfig   `json:"defaultUser"`
	Port           int           `json:"port"`
	IP             string        `json:"ip"`
	Log            string        `json:"log"`
	BaseUrl        string        `json:"baseUrl"`
	PrefixUrl      string        `json:"prefixUrl"`
	RefreshSeconds int           `json:"configRamRefreshSeconds"`
	*CaptchaConfig `json:"captchaConfig"`
	*Auth          `json:"auth"`
	*PreviewConf   `json:"preview"`
	updateLock     *sync.RWMutex `json:"-"`
	needSave       int32         `json:"-"`
}

// Auth settings.
type PreviewConf struct {
	//enable preview generating by call .sh
	AllowGeneratePreview bool `json:"allowGeneratePreview"`
	Threads              int  `json:"threads"`
	FirstRun             bool `json:"firstRun"`
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
	cfg.refreshRam()
	cfg.updateLock = new(sync.RWMutex)
	config = cfg

	for _, u := range cfg.Users {
		u.sortShares()
	}
}

func (cfg *GlobalConfig) StartMonitor() {
	go func() {
	Start:
		cfgCounter := atomic.SwapInt32(&cfg.needSave, 0)
		if cfgCounter != 0 {
			cfg.updateLock.Lock()
			jsonData, err := json.MarshalIndent(cfg, "", "    ")
			if err != nil {
				fmt.Println(err)
			} else {
				err = ioutil.WriteFile(FileName, jsonData, 0644)
			}
			cfg.updateLock.Unlock()
		}
		time.Sleep(time.Duration(cfg.RefreshSeconds) * time.Second)
		goto Start
	}()
}

func (cfg *GlobalConfig) Store() {
	cfg.needSave = atomic.AddInt32(&cfg.needSave, 1)
}

func (cfg *GlobalConfig) refreshRam() {
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
	if strings.EqualFold(username, cfg.DefaultUser.Username) {
		return cfg.DefaultUser.copyUser(), true
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

	for _, ipAuth := range cfg.DefaultUser.IpAuth {
		if strings.EqualFold(ip, ipAuth) {
			return cfg.DefaultUser.copyUser(), true
		}
	}

	res, ok := usersRam[ip]
	if !ok {
		return nil, ok
	}

	return res.copyUser(), ok
}

func (cfg *GlobalConfig) Gets(defUser bool) (res []*UserConfig) {
	cfg.lockR()
	defer cfg.unlockR()
	res = make([]*UserConfig, len(cfg.Users))
	for i, u := range cfg.Users {
		res[i] = u.copyUser()
	}
	if defUser {
		res = append(res, cfg.DefaultUser)
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
	cfg.refreshRam()

	return nil
}
func (cfg *GlobalConfig) Update(u *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()

	if strings.EqualFold(u.Username, cfg.DefaultUser.Username) {
		cfg.DefaultUser = u.copyUser()
		return nil
	}

	_, exists := usersRam[u.Username]
	if !exists {
		return errors.New("User does not exists " + u.Username)
	}

	i := cfg.getUserIndex(u.Username)

	if strings.EqualFold(cfg.Users[i].Username, u.Username) {
		cfg.Users[i] = cfg.Users[i].copyUser()
		cfg.refreshRam()
	}
	return nil
}

func (cfg *GlobalConfig) UpdateUsers(users []*UserConfig, defUser *UserConfig) error {
	cfg.lock()
	defer cfg.unlock()
	if len(users) > 0 {
		cfg.Users = make([]*UserConfig, len(users))
		for i, u := range users {
			cfg.Users[i] = u.copyUser()
		}
	}
	if defUser != nil {
		cfg.DefaultUser = defUser.copyUser()
	}
	cfg.refreshRam()
	return nil
}

func (cfg *GlobalConfig) Delete(username string) (error) {
	cfg.lock()
	defer cfg.unlock()

	i := cfg.getUserIndex(username)
	if strings.EqualFold(cfg.Users[i].Username, username) {
		cfg.Users = append(cfg.Users[:i], cfg.Users[i+1:]...)

	}
	cfg.refreshRam()

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

func (cfg *GlobalConfig) CopyConfig() (res *GlobalConfig) {
	cfg.lockR()
	defer cfg.unlockR()
	res = &GlobalConfig{
		Users:          cfg.Gets(false),
		DefaultUser:    cfg.DefaultUser.copyUser(),
		RefreshSeconds: cfg.RefreshSeconds,
		BaseUrl:        cfg.BaseUrl,
		PrefixUrl:      cfg.PrefixUrl,
		Port:           cfg.Port,
		IP:             cfg.IP,
		Log:            cfg.Log,
		CaptchaConfig:  cfg.CopyCaptchaConfig(),
		Auth:           cfg.CopyAuth(),
		PreviewConf:    &PreviewConf{AllowGeneratePreview: cfg.AllowGeneratePreview, Threads: cfg.Threads},
	}
	return res
}
func (cfg *GlobalConfig) UpdateConfig(u *GlobalConfig) {
	cfg.lock()
	defer cfg.unlock()
	cfg.RefreshSeconds = u.RefreshSeconds
	cfg.BaseUrl = u.BaseUrl
	cfg.Port = u.Port
	cfg.PrefixUrl = u.PrefixUrl
	cfg.IP = u.IP
	cfg.Log = u.Log
	cfg.Users = u.Gets(false)
	cfg.DefaultUser = u.DefaultUser.copyUser()
	cfg.Auth = u.CopyAuth()
	cfg.CaptchaConfig = u.CopyCaptchaConfig()
}

func (cfg *GlobalConfig) SetKey(k []byte) {
	cfg.Auth.Key = base64.StdEncoding.EncodeToString(k)
}
func (cfg *GlobalConfig) lock() {
	cfg.needSave = atomic.AddInt32(&cfg.needSave, 1)
	cfg.updateLock.Lock()
}
func (cfg *GlobalConfig) unlock() {
	cfg.updateLock.Unlock()
}

func (cfg *GlobalConfig) lockR() {
	cfg.updateLock.RLock()
}
func (cfg *GlobalConfig) unlockR() {
	cfg.updateLock.RUnlock()
}
