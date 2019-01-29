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
var fileName = "filebrowser.json"

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
	CaptchaConfig  `json:"captchaConfig"`
	Auth           `json:"auth"`
	updateLock     *sync.RWMutex `json:"-"`
	needSave       int32         `json:"-"`
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

type CaptchaConfig struct {
	Host   string `json:"host"`
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

func (cfg *GlobalConfig) Verify() {

	//todo
}
func (cfg *GlobalConfig) ReadConfigFile(file string) {
	fileName = file
	// Open our jsonFile
	jsonFile, err := os.Open(file)
	defer jsonFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &cfg)
	cfg.refreshRam()
	cfg.updateLock = new(sync.RWMutex)
	config = cfg
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
				err = ioutil.WriteFile(fileName, jsonData, 0644)
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

	for i := 0; i < len(cfg.Users); i++ {
		//index usernames
		usersRam[cfg.Users[i].Username] = cfg.Users[i]
		for j := 0; j < len(cfg.Users[i].IpAuth); j++ {
			//index ips as well
			usersRam[cfg.Users[i].IpAuth[j]] = cfg.Users[i]
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
	ip = strings.Split(ip, ":")[0]

	for i := 0; i < len(cfg.DefaultUser.IpAuth); i++ {
		if strings.EqualFold(ip, cfg.DefaultUser.IpAuth[i]) {
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
	res = make([]*UserConfig, len(cfg.Users))
	for i := 0; i < len(cfg.Users); i++ {
		res[i] = cfg.Users[i].copyUser()
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
		for i := 0; i < len(users); i++ {
			cfg.Users[i] = users[i].copyUser()
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
	for i := 0; i < len(cfg.Users); i++ {
		if strings.EqualFold(cfg.Users[i].Username, userName) {
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
