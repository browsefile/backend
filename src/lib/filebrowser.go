package lib

import (
	"crypto/rand"
	"errors"
	"github.com/GeertJohan/go.rice"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/browsefile/backend/src/lib/preview"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"
)

const (
	// Version is the current File Browser version.
	Version        = "(untracked)"
	MosaicViewMode = "mosaic"
)

var (
	ErrExist              = errors.New("the resource already exists")
	ErrNotExist           = errors.New("the resource does not exist")
	ErrEmptyRequest       = errors.New("request body is empty")
	ErrEmptyPassword      = errors.New("password is empty")
	ErrEmptyUsername      = errors.New("username is empty")
	ErrEmptyScope         = errors.New("scope is empty")
	ErrWrongDataType      = errors.New("wrong data type")
	ErrInvalidUpdateField = errors.New("invalid field to update")
	ErrInvalidOption      = errors.New("invalid option")
)

// ReCaptcha settings.
type ReCaptcha struct {
	Host   string
	Key    string
	Secret string
}

// FileBrowser is a file manager instance. It should be creating using the
// 'New' function and not directly.
type FileBrowser struct {
	// The static assets.
	Assets *rice.Box
	// PrefixURL is a part of the URL that is already trimmed from the request URL before it
	// arrives to our handlers. It may be useful when using File Browser as a middleware
	// such as in caddy-filemanager plugin. It is only useful in certain situations.
	PrefixURL string
	// BaseURL is the path where the GUI will be accessible. It musn't end with
	// a trailing slash and mustn't contain PrefixURL, if set. It shouldn't be
	// edited directly. Use SetBaseURL.
	BaseURL string
	// ReCaptcha host, key and secret.
	ReCaptcha *ReCaptcha
	// Global stylesheet.
	CSS string
	// NewFS should build a new file system for a given path.
	NewFS FSBuilder
	//generates preview
	Pgen   *preview.PreviewGen
	Config *config.GlobalConfig
}

// FSBuilder is the File System Builder.
type FSBuilder func(scope string) FileSystem

// Setup loads the configuration from the database and configures
// the Assets and the Cron job. It must always be run after
// creating a File Browser object.
func (fb *FileBrowser) Setup() (bool, error) {
	needUpdate := false
	// Creates a new File Browser instance with the Users
	// map and Assets box.
	fb.Assets = rice.MustFindBox("../../frontend/dist")

	// Tries to get the encryption key from the database.
	// If it doesn't exist, create a new one of 256 bits.
	_, err := fb.Config.GetKeyBytes()
	if err != nil || err == ErrNotExist {
		var bytes []byte
		bytes, err = GenerateRandomBytes(64)
		if err != nil {
			return needUpdate, err
		}

		needUpdate = true
		fb.Config.SetKey(bytes)
	}
	users := fb.Config.GetUsers()
	for _, u := range users {
		if u.FirstRun {
			u.FirstRun = false
			needUpdate = true
			u.Password, err = HashPassword(u.Password)
			if err != nil {
				log.Println(err)
			}
		}
	}

	if needUpdate {
		fb.Config.UpdateUsers(users)
	}
	fb.Pgen = new(preview.PreviewGen)
	fb.Pgen.Setup(fb.Config.Threads)

	//generate all previews for the first run
	if fb.Config.FirstRun {
		fb.Config.FirstRun = false
		needUpdate = true
		go func() {
			allUs := fb.Config.GetUsers()
			for i := 0; i < len(allUs); i++ {
				u := allUs[i]
				fb.Pgen.ProcessPath(u.Scope, u.PreviewScope)
			}
		}()
	}

	return needUpdate, nil
}
func (c *Context) GenPreview(out string) {
	if c.Config.AllowGeneratePreview {
		_, t := fileutils.GetBasedOnExtensions(c.File.Name)
		if t == "image" || t == "video" {
			c.Pgen.Process(c.Pgen.GetDefaultData(c.File.Path, out, t))
		}
	}
}

// RootURL returns the actual URL where
// File Browser interface can be accessed.
func (m FileBrowser) RootURL() string {
	return m.PrefixURL + m.BaseURL
}

// SetPrefixURL updates the prefixURL of a File
// Manager object.
func (m *FileBrowser) SetPrefixURL(url string) {
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")
	url = "/" + url
	m.PrefixURL = strings.TrimSuffix(url, "/")
}

// SetBaseURL updates the baseURL of a File Browser
// object.
func (m *FileBrowser) SetBaseURL(url string) {
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")
	url = "/" + url
	m.BaseURL = strings.TrimSuffix(url, "/")
}

// DefaultUser is used on New, when no 'base' user is provided.
var DefaultUser = UserModel{
	UserConfig: &config.UserConfig{
		AllowEdit:    true,
		AllowNew:     true,
		LockPassword: false,
		Admin:        true,
		Locale:       "",
		Scope:        "./scope",
		ViewMode:     "mosaic",
		PreviewScope: "./preview",
	},
	FileSystem:        fileutils.Dir("."),
	FileSystemPreview: fileutils.Dir("."),
}

// FileSystem is the interface to work with the file system.
type FileSystem interface {
	Mkdir(name string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	RemoveAll(name string) error
	Rename(oldName, newName string) error
	Stat(name string) (os.FileInfo, error)
	Copy(src, dst string) error
}

type UserModel struct {
	*config.UserConfig
	ID string `json:"ID"`
	// FileSystem is the virtual file system the user has access.
	FileSystem FileSystem `json:"-"`
	// FileSystem is the virtual file system the user has access, uses to store previews.
	FileSystemPreview FileSystem `json:"-"`
}

// Context contains the needed information to make handlers work.
type Context struct {
	*FileBrowser
	User *UserModel
	File *File
	// On API handlers, Router is the APi handler we want.
	Router string
	//indicate that requested preview
	PreviewType string
	//return files list by recursion
	IsRecursive bool
	//indicate about share request
	ShareUser string
}

// HashPassword generates an hash from a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with an hash to check if they match.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an fm.Error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}
