package cnst

import (
	"net/http"
	"os"
)

//header keys and request url params
var (
	H_XAUTH        = "X-Auth"
	P_PREVIEW_TYPE = "previewType"
	P_EXSHARE      = "exshare"
)
var (
	// Version is the current File Browser version.
	Version        = "(untracked)"
	MosaicViewMode = "mosaic"
	CommitSHA      = ""
)

//settings
const (
	FilePath1      = "browsefile.json"
	FilePath2      = "/etc/browsefile.json"
	GUEST          = "guest"
	WEB_DAV_URL    = "/" + WEB_DAV_FOLDER
	WEB_DAV_FOLDER = "wd"
	//default permission for paths creation
	PERM_DEFAULT = 0765
)

//mime types
const (
	VIDEO = "video"
	IMAGE = "image"
	AUDIO = "audio"
	PDF   = "pdf"
	TEXT  = "text"
	BLOB  = "blob"
)

//routes
const (
	R_SEARCH   = 1
	R_SETTINGS = 2
	R_USERS    = 3
	R_RESOURCE = 4
	R_DOWNLOAD = 5
	R_SHARES   = 7
	R_PLAYLIST = 8
)

var MIME_EXT = [][]string{{
	".jpg", ".png", ".jpeg", ".tiff", ".tif", ".bmp",
	".gif", ".eps", ".raw", ".cr2", ".nef", ".orf", ".sr2",
}, {
	".3gp", ".3g2", ".asf", ".wma", ".wmv",
	".avi", ".divx", ".f4v", ".evo", ".flv",
	".mkv", ".mk3d", ".mka", ".mks", ".webm",
	".mcf", ".mp4", ".mpg", ".mpeg", ".m2p",
	".ps", ".ts", ".m2ts", ".mxf",
	".mov", ".qt", ".rmvb", ".vob",
}, {
	".aa", ".aac", ".mp3", ".aiff", ".amr", ".act", ".aax",
	".au", ".awb", ".flac", ".m4a", ".m4b", ".m4p", ".ra", ".rm", ".wav",
	".alac", ".ogg",
}, {
	".ad", ".ada", ".adoc", ".asciidoc",
	".bas", ".bash", ".bat",
	".c", ".cc", ".cmd", ".conf", ".cpp", ".cr", ".cs", ".css", ".csv",
	".d",
	".f", ".f90",
	".h", ".hh", ".hpp", ".htaccess", ".html",
	".ini",
	".java", ".js", ".json",
	".markdown", ".md", ".mdown", ".mmark",
	".nim",
	".php", ".pl", ".ps1", ".py", ".go",
	".rss", ".rst", ".rtf",
	".sass", ".scss", ".sh", ".sty",
	".tex", ".tml", ".toml", ".txt",
	".vala", ".vapi",
	".xml",
	".yaml", ".yml",
	"caddyfile",
}}

// ErrorToHTTP converts errors to HTTP Status Code.
func ErrorToHTTP(err error, gone bool) int {
	switch {
	case err == nil:
		return http.StatusOK
	case os.IsPermission(err):
		return http.StatusForbidden
	case os.IsNotExist(err):
		if !gone {
			return http.StatusNotFound
		}

		return http.StatusGone
	case os.IsExist(err):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
