package lib

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib/fileutils"
	"github.com/maruel/natural"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FitFilter func(name, p string) bool

// File contains the information about a particular file or directory.
type File struct {
	// Indicates the Kind of view on the front-end (Listing, editor or preview).
	Kind string `json:"kind"`
	// The name of the file.
	Name string `json:"name"`
	// The Size of the file.
	Size int64 `json:"size"`
	// The absolute URL.
	URL string `json:"url"`
	// The extension of the file.
	Extension string `json:"extension"`
	// The last modified time.
	ModTime time.Time `json:"modified"`
	// The File Mode.
	Mode os.FileMode `json:"mode"`
	// Indicates if this file is a directory.
	IsDir bool `json:"isDir"`
	// Absolute path.
	Path string `json:"-"`
	// Relative path to user's virtual File System.
	VirtualPath string `json:"-"`
	// Indicates the file content type: video, text, image, music or blob.
	Type string `json:"type"`
	// Stores the content of a text file.
	Content string `json:"content,omitempty"`

	Checksums map[string]string `json:"checksums,omitempty"`
	*Listing  `json:",omitempty"`

	Language string `json:"language,omitempty"`
}

// A Listing is the context used to fill out a template.
type Listing struct {
	// The items (files and folders) in the path.
	Items []*File `json:"items"`
	// The number of directories in the Listing.
	NumDirs int `json:"numDirs"`
	// The number of files (items that aren't directories) in the Listing.
	NumFiles int `json:"numFiles"`
	// Which sorting order is used.
	Sort string `json:"sort"`
	// And which order.
	Order string `json:"order"`
	//indicator to the frontend, to prevent request previews
	AllowGeneratePreview bool `json:"allowGeneratePreview"`
}

// build correct path, and replace user in context in case external share
func ResolvePaths(c *Context, url string) (p, previewPath, urlPath string, err error) {
	if c.IsShare {
		if c.IsExternalShare() {
			itm, usr := c.Config.GetExternal(c.RootHash)
			if itm == nil {
				return "", "", "", cnst.ErrNotExist
			}
			if !itm.IsAllowed(c.User.Username) {
				return "", "", "", cnst.ErrShareAccess
			}
			c.User = ToUserModel(usr, c.Config)
			/*urlPath = itm.Path
			urlString = urlPath*/
			p, previewPath = c.GetUserHomePath(), c.GetUserPreviewPath()
			//if share root listing

			if len(url) == 1 {
				urlPath = itm.Path
			} else {
				urlPath = itm.Path + url
			}

		} else {
			p, previewPath = c.GetUserSharesPath(), c.Config.GetSharePreviewPath(url)
			urlPath = url
		}

	} else {
		p, previewPath = c.GetUserHomePath(), c.GetUserPreviewPath()
		urlPath = url
	}
	return
}

// MakeInfo gets the file information and, in case of error, returns the
// respective HTTP error code
func MakeInfo(urlPath, urlString string, c *Context) (*File, error) {
	p, previewPath, urlPath2, err := ResolvePaths(c, urlPath)
	urlPath = urlPath2

	info, err, path := fileutils.GetFileInfo(p, urlPath)

	//create user paths if not exists
	if os.IsNotExist(err) {
		err = os.MkdirAll(p, 0775)
		os.MkdirAll(previewPath, 0775)
		fileutils.ModPermission(c.User.UID, c.User.GID, p)
		if err != nil {
			return nil, err
		}
		info, err, path = fileutils.GetFileInfo(p, urlPath)
	}

	i := &File{
		URL:         urlString,
		VirtualPath: urlPath,
		Path:        path,
	}

	if err != nil {
		return i, err
	}

	i.Name = info.Name()
	i.ModTime = info.ModTime()
	i.Mode = info.Mode()
	i.IsDir = info.IsDir()
	i.Size = info.Size()
	i.Extension = filepath.Ext(i.Name)

	if i.IsDir && !strings.HasSuffix(i.URL, "/") {
		i.URL += "/"
	}

	return i, nil
}
func (i *File) MakeListing(c *Context, fitFilter FitFilter) (files []os.FileInfo, paths []string, err error) {
	isExternal := c.IsExternalShare()

	if c.IsRecursive {
		files = make([]os.FileInfo, 0, 500)
		paths = make([]string, 0, 500)
		isShare := c.IsShare
		var p string
		if isShare && !isExternal {
			p = c.GetUserSharesPath()
		} else {
			p = c.GetUserHomePath()
		}
		files, paths = i.listRecurs(c, fitFilter, filepath.Join(p, i.VirtualPath), files, paths)
	} else {
		var f *os.File
		var err error
		if isExternal {
			f, err = c.User.FileSystem.OpenFile(i.VirtualPath, os.O_RDONLY, 0, c.User.UID, c.User.GID)
			//replace original user
			usr, _ := c.Config.GetByUsername("guest")
			c.User = ToUserModel(usr, c.Config)

		} else if c.IsShare {
			f, err = c.User.FileSystemShares.OpenFile(i.VirtualPath, os.O_RDONLY, 0, c.User.UID, c.User.GID)
		} else {
			f, err = c.User.FileSystem.OpenFile(i.VirtualPath, os.O_RDONLY, 0, c.User.UID, c.User.GID)
		}

		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		// Reads the directory and gets the information about the files.
		files, err = f.Readdir(-1)
		if err != nil {
			return nil, nil, err
		}
	}
	return files, paths, nil

}
func (i *File) listRecurs(c *Context, fitFilter FitFilter, path string, files []os.FileInfo, paths []string) ([]os.FileInfo, []string) {
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if c.IsShare && !c.IsExternalShare() {
				f2, _ := filepath.EvalSymlinks(path)
				info, _ = os.Stat(f2)
				if info.IsDir() {
					fls, e := ioutil.ReadDir(f2)
					if e != nil {
						return nil
					}
					for _, f := range fls {
						if f.IsDir() {
							files, paths = i.listRecurs(c, fitFilter, filepath.Join(f2, f.Name()), files, paths)

						} else {
							path = strings.TrimPrefix(f2, c.Config.FilesPath)
							if fitFilter != nil && fitFilter(f.Name(), path) ||
								fitFilter == nil {
								//cut files from path
								arr := strings.SplitN(path, "/", 4)
								paths = append(paths, filepath.Join("/", arr[1], arr[3], f.Name()))
								files = append(files, f)
							}
						}
					}
					return nil
				}

			} else if fitFilter != nil {
				path = strings.TrimPrefix(path, c.GetUserHomePath())

				if fitFilter != nil && fitFilter(info.Name(), path) || fitFilter == nil {
					if c.IsExternalShare() {
						//replace root share folder with rootHash
						path = "/" + strings.SplitN(path, "/", 3)[2]
					} else {
						path = strings.TrimPrefix(path, c.GetUserHomePath())
					}
					files = append(files, info)
					paths = append(paths, path)
				}
			}

			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return files, paths
}

// GetListing gets the information about a specific directory and its files.
func (i *File) GetListing(c *Context, fitFilter FitFilter) error {
	// GetUsers the directory information using the Virtual File System of
	// the user configuration.
	var (
		files               []os.FileInfo
		paths               []string
		fileinfos           []*File
		dirCount, fileCount int
		// Absolute URL
		fUrl url.URL
	)
	files, paths, err := i.MakeListing(c, fitFilter)
	if err != nil {
		log.Println("file: ", err)
		return err
	}
	baseurl, err := url.PathUnescape(i.URL)
	if err != nil {
		return err
	}

	isShare := c.IsShare
	isExternal := c.IsExternalShare()
	if isShare && c.IsRecursive {
		isShare = false
	}
	for ind, f := range files {
		name := f.Name()

		if strings.HasPrefix(f.Mode().String(), "L") {
			// It's a symbolic link. We try to follow it. If it doesn't work,
			// we stay with the link information instead if the target's.
			info, err := os.Stat(f.Name())
			if err == nil {
				f = info
			}
		}
		if isShare && !c.IsRecursive && !isExternal {
			p := filepath.Join(i.Path, name)
			f2, _ := filepath.EvalSymlinks(p)
			f, _ = os.Stat(f2)
		}

		if f.IsDir() {
			name += "/"
			dirCount++
		} else {
			fileCount++
		}

		if c.IsRecursive {

			if f.IsDir() {
				fUrl = url.URL{Path: baseurl}
			} else {
				fUrl = url.URL{Path: paths[ind]}
			}

		} else {
			fUrl = url.URL{Path: baseurl + name}
		}

		i := &File{
			Name:      f.Name(),
			Size:      f.Size(),
			ModTime:   f.ModTime(),
			Mode:      f.Mode(),
			IsDir:     f.IsDir(),
			URL:       fUrl.String(),
			Extension: filepath.Ext(name),
		}
		i.SetFileType(false)

		if fitFilter != nil && !c.IsRecursive {
			if fitFilter(i.Name, fUrl.Path) {
				fileinfos = append(fileinfos, i)
			} else {
				if f.IsDir() {
					dirCount--
				} else {
					fileCount--
				}

			}

		} else {
			fileinfos = append(fileinfos, i)
		}
	}

	i.Listing = &Listing{
		Items:    fileinfos,
		NumDirs:  dirCount,
		NumFiles: fileCount,
	}

	return nil
}

// SetFileType obtains the mimetype and converts it to a simple
// type nomenclature.
func (f *File) SetFileType(checkContent bool) {
	if len(f.Type) > 0 || f.IsDir {
		return
	}
	var isOk bool
	isOk, f.Type = fileutils.GetBasedOnExtensions(f.Extension)
	// Tries to get the file mimetype using its extension.
	if !isOk && checkContent {
		log.Println("Can't detect file type, based on extension ", f.Name)
		return

	}
}

// Checksum retrieves the checksum of a file.
func (i *File) Checksum(algo string) error {
	if i.IsDir {
		return cnst.ErrIsDirectory
	}

	if i.Checksums == nil {
		i.Checksums = make(map[string]string)
	}

	file, err := os.Open(i.Path)
	if err != nil {
		return err
	}

	defer file.Close()

	var h hash.Hash

	switch algo {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return cnst.ErrInvalidOption
	}

	_, err = io.Copy(h, file)
	if err != nil {
		return err
	}

	i.Checksums[algo] = hex.EncodeToString(h.Sum(nil))
	return nil
}

// CanBeEdited checks if the extension of a file is supported by the editor
func (i File) CanBeEdited() bool {
	return i.Type == cnst.TEXT
}

// ApplySort applies the sort order using .Order and .Sort
func (l Listing) ApplySort() {
	// Check '.Order' to know how to sort
	if l.Order == "desc" {
		switch l.Sort {
		case "name":
			sort.Sort(sort.Reverse(byName(l)))
		case "size":
			sort.Sort(sort.Reverse(bySize(l)))
		case "modified":
			sort.Sort(sort.Reverse(byModified(l)))
		default:
			// If not one of the above, do nothing
			return
		}
	} else { // If we had more Orderings we could add them here
		switch l.Sort {
		case "name":
			sort.Sort(byName(l))
		case "size":
			sort.Sort(bySize(l))
		case "modified":
			sort.Sort(byModified(l))
		default:
			sort.Sort(byName(l))
			return
		}
	}
}

// Implement sorting for Listing
type byName Listing
type bySize Listing
type byModified Listing

// By Name
func (l byName) Len() int {
	return len(l.Items)
}

func (l byName) Swap(i, j int) {
	l.Items[i], l.Items[j] = l.Items[j], l.Items[i]
}

// Treat upper and lower case equally
func (l byName) Less(i, j int) bool {
	if l.Items[i].IsDir && !l.Items[j].IsDir {
		return true
	}

	if !l.Items[i].IsDir && l.Items[j].IsDir {
		return false
	}

	return natural.Less(strings.ToLower(l.Items[j].Name), strings.ToLower(l.Items[i].Name))
}

// By Size
func (l bySize) Len() int {
	return len(l.Items)
}

func (l bySize) Swap(i, j int) {
	l.Items[i], l.Items[j] = l.Items[j], l.Items[i]
}

const directoryOffset = -1 << 31 // = math.MinInt32
func (l bySize) Less(i, j int) bool {
	iSize, jSize := l.Items[i].Size, l.Items[j].Size
	if l.Items[i].IsDir {
		iSize = directoryOffset + iSize
	}
	if l.Items[j].IsDir {
		jSize = directoryOffset + jSize
	}
	return iSize < jSize
}

// By Modified
func (l byModified) Len() int {
	return len(l.Items)
}

func (l byModified) Swap(i, j int) {
	l.Items[i], l.Items[j] = l.Items[j], l.Items[i]
}

func (l byModified) Less(i, j int) bool {
	iModified, jModified := l.Items[i].ModTime, l.Items[j].ModTime
	return iModified.Sub(jModified) < 0
}
