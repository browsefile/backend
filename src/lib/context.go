package lib

import (
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib/utils"
	"net/http"
	"net/url"
	"strings"
)

//used to filter out specific files by name and path
type FitFilter func(name, p string) bool

// Context contains the needed information to make handlers work.
type Context struct {
	*FileBrowser
	User *UserModel
	File *File
	// On API handlers, Router is the APi handler we want.
	Router int
	*Params
	FitFilter
	Rendered bool
}

//params in URL request
type Params struct {
	//indicate that requested preview
	PreviewType string
	//return files list by recursion
	IsRecursive bool
	//indicate about share request, nformation about share metadata like list, my-meta.
	ShareType    string
	SearchString string
	/*
		external share item root dir hash,
		note: as a result, cut owner from files paths, and replaces with rootHash(hash value of specific shareItem)
	*/
	IsExternal bool
	//download type, zip or playlist m3u8
	Algo string
	//download multiple files
	FilePaths []string

	Auth string

	Checksum string

	Inline bool
	// playlist & search file mime types, true if any was specified at request url, uses in FitFilter type
	Audio bool
	Image bool
	Video bool
	Pdf   bool
	//search query
	Query url.Values
	//override existing file
	Override bool
	// used in resource patch requests type
	Destination string
	Action      string

	Sort  string
	Order string
	//is share request
	IsShare bool
	//requestURL
	URL    string
	Method string
	RESP   http.ResponseWriter
	REQ    *http.Request
}

//cut user home path from for non download routes
func (c *Context) CutPath(path string) string {
	if c.Router != cnst.R_DOWNLOAD {
		if c.IsShare {
			if c.IsExternal {
				path = strings.TrimPrefix(path, c.GetUserSharexPath())
			} else {
				path = strings.TrimPrefix(path, c.GetUserSharesPath())
			}

		} else {
			path = strings.TrimPrefix(path, c.GetUserHomePath())
		}
	}
	return path

}
func (c *Context) GetUserHomePath() string {
	return c.Config.GetUserHomePath(c.User.Username)
}
func (c *Context) GetUserPreviewPath() string {
	return c.Config.GetUserPreviewPath(c.User.Username)
}
func (c *Context) GetUserSharesPath() string {
	return c.Config.GetUserSharesPath(c.User.Username)
}
func (c *Context) GetUserSharexPath() string {
	return c.Config.GetUserSharexPath(c.User.Username)
}
func (c *Context) ResolvePathContext(i *File) (p string, fs FileSystem) {
	if c.IsExternal {
		fs = c.User.FileSystemSharex
		p = c.GetUserSharexPath()
	} else if c.IsShare {
		fs = c.User.FileSystemShares
		p = c.GetUserSharesPath()
	} else {
		fs = c.User.FileSystem
		p = c.GetUserHomePath()
	}
	return
}

func (c *Context) GenPreview(out string) {
	if len(c.Config.ScriptPath) > 0 {
		_, t := utils.GetFileType(c.File.Name)
		if t == cnst.IMAGE || t == cnst.VIDEO {
			c.Pgen.Process(c.Pgen.GetDefaultData(c.File.Path, out, t))
		}
	}
}

func (c *Context) GenSharesPreview(out string) {
	if len(c.Config.ScriptPath) > 0 {

		_, t := utils.GetFileType(c.File.Name)
		if t == cnst.IMAGE || t == cnst.VIDEO {
			c.Pgen.Process(c.Pgen.GetDefaultData(c.File.Path, out, t))

		}

	}
}

func (c *Context) GetAuthConfig() *config.ListenConf {
	isTls := c.REQ.TLS != nil
	var cfgM *config.ListenConf
	if isTls {
		cfgM = c.Config.Tls
	} else {
		cfgM = c.Config.Http

	}
	return cfgM
}

// MakeInfo gets the file information, and replace user in context in case share rquest
func (c *Context) MakeInfo() (*File, error) {
	p, _, err := c.ResolveContextUser()
	info, err, path := utils.GetFileInfo(p, c.URL)
	if err != nil {
		return nil, err
	}
	i := &File{
		URL:         c.URL,
		VirtualPath: utils.SlashClean(c.URL),
		Path:        path,
		Name:        info.Name(),
		IsDir:       info.IsDir(),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
	}

	if i.IsDir && !strings.HasSuffix(i.URL, "/") {
		i.URL += "/"
	}
	i.URL = url.PathEscape(i.URL)

	return i, nil
}

// build correct path, and replace user in context in case external share
func (c *Context) ResolveContextUser() (p, previewPath string, err error) {
	if c.IsExternal {
		var h string

		previewPath, h = c.Config.GetSharePreviewPath(c.URL, true)

		itm, usr := c.Config.GetExternal(h)
		if itm == nil {
			return "", "", cnst.ErrNotExist
		}
		if !itm.IsAllowed(c.User.Username) {
			return "", "", cnst.ErrShareAccess
		}
		c.User = ToUserModel(usr, c.Config)
		p = c.GetUserSharexPath()

	} else if c.IsShare {
		previewPath, _ = c.Config.GetSharePreviewPath(c.URL, false)
		p = c.GetUserSharesPath()

	} else {
		p, previewPath = c.GetUserHomePath(), c.GetUserPreviewPath()
	}
	return
}
