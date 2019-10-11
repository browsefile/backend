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
	//Extension string `json:"extension"`
	// The last modified time.
	ModTime time.Time `json:"modified"`
	// The File Mode.
	//Mode os.FileMode `json:"mode"`
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
func ResolvePaths(c *Context) (p, previewPath, urlPath string, err error) {
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
			p, previewPath = c.GetUserHomePath(), c.GetUserPreviewPath()
			//if share root listing

			if len(c.URL) == 1 {
				urlPath = itm.Path
			} else {
				urlPath = itm.Path + c.URL
			}

		} else {
			p, previewPath = c.GetUserSharesPath(), c.Config.GetSharePreviewPath(c.URL)
			urlPath = c.URL
		}

	} else {
		p, previewPath = c.GetUserHomePath(), c.GetUserPreviewPath()
		urlPath = c.URL
	}
	return
}

// MakeInfo gets the file information
func MakeInfo(c *Context) (*File, error) {
	p, _, urlPath2, err := ResolvePaths(c)
	c.URL = urlPath2

	info, err, path := fileutils.GetFileInfo(p, c.URL)
	if err != nil {
		return nil, err
	}
	i := &File{
		URL:         c.URLString,
		VirtualPath: c.URL,
		Path:        path,
		Name:        info.Name(),
		IsDir:       info.IsDir(),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
	}

	if i.IsDir && !strings.HasSuffix(i.URL, "/") {
		i.URL += "/"
	}

	return i, nil
}

//recursively fetch share/file paths
func (i *File) GetListing(c *Context) (files []os.FileInfo, paths []string, err error) {
	isExternal := c.IsExternalShare()

	if c.IsRecursive {

		var p string
		if c.IsShare && !isExternal {
			p = c.GetUserSharesPath()
		} else {
			p = c.GetUserHomePath()
		}
		files, paths = i.listRecurs(c, filepath.Join(p, i.VirtualPath))
	} else {
		var f *os.File
		var err error
		if isExternal {
			f, err = c.User.FileSystem.OpenFile(i.VirtualPath, os.O_RDONLY, 0, c.User.UID, c.User.GID)
			//replace original user/owner
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
func (i *File) listRecurs(c *Context, path string) (files []os.FileInfo, paths []string) {
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if c.IsShare && !c.IsExternalShare() {
				//get files from current user shares folder
				f2, _ := filepath.EvalSymlinks(path)
				info, _ = os.Stat(f2)
				if info.IsDir() {
					fls, e := ioutil.ReadDir(f2)
					if e != nil {
						return nil
					}
					for _, f := range fls {
						if f.IsDir() {
							fr, pr := i.listRecurs(c, filepath.Join(f2, f.Name()))
							files = append(files, fr...)
							paths = append(paths, pr...)

						} else {
							//ignore path cut for download, full path required for download handler
							if c.Router != cnst.R_DOWNLOAD {
								path = strings.TrimPrefix(f2, c.Config.FilesPath)
							}
							if c.FitFilter != nil && c.FitFilter(f.Name(), path) ||
								c.FitFilter == nil {
								//cut files from path

								if c.Router != cnst.R_DOWNLOAD {
									arr := strings.SplitN(path, "/", 4)
									paths = append(paths, filepath.Join("/", arr[1], arr[3], f.Name()))
								} else {
									paths = append(paths, path)
								}
								files = append(files, f)
							}
						}
					}
					return nil
				}

			} else {
				if c.Router != cnst.R_DOWNLOAD {
					path = strings.TrimPrefix(path, c.GetUserHomePath())
				}
				if c.FitFilter != nil && c.FitFilter(info.Name(), path) || c.FitFilter == nil {
					if c.IsExternalShare() {
						//replace root share folder with rootHash
						path = "/" + strings.SplitN(path, "/", 3)[2]
					} else if c.Router != cnst.R_DOWNLOAD {
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

// ProcessList generate metainfo about dir/files
func (i *File) ProcessList(c *Context) error {
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
	files, paths, err := i.GetListing(c)
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

		//resolve share symlink
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
		//take path from recursive fetch, otherwise use current
		if c.IsRecursive {

			if f.IsDir() {
				fUrl = url.URL{Path: baseurl}
			} else {
				fUrl = url.URL{Path: paths[ind]}
			}

		} else {
			fUrl = url.URL{Path: baseurl + name}
		}

		fI := &File{
			Name:    f.Name(),
			Size:    f.Size(),
			ModTime: f.ModTime(),
			IsDir:   f.IsDir(),
			URL:     fUrl.String(),
		}
		fI.SetFileType(false)

		if c.FitFilter != nil && !c.IsRecursive {
			if c.FitFilter(fI.Name, fUrl.Path) {
				fileinfos = append(fileinfos, fI)
			} else {
				if f.IsDir() {
					dirCount--
				} else {
					fileCount--
				}
			}

		} else {
			fileinfos = append(fileinfos, fI)
		}
	}

	i.Listing = &Listing{
		Items:    fileinfos,
		NumDirs:  dirCount,
		NumFiles: fileCount,
	}
	if i.Listing.Items == nil {
		i.Listing.Items = []*File{}
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
	isOk, f.Type = fileutils.GetBasedOnExtensions(filepath.Ext(f.Name))
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
