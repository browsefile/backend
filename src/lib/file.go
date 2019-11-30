package lib

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib/utils"
	"github.com/maruel/natural"
	"hash"
	"io"
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

//recursively fetch share/file paths
func (i *File) GetListing(c *Context) (files []os.FileInfo, paths []string, err error) {
	p, fs := c.ResolvePathContext(i)
	//fetch all files
	if c.IsRecursive {
		files, paths = i.listRecurs(c, filepath.Join(p, i.VirtualPath))
	} else {
		//only list content
		inf, err := fs.Stat(i.VirtualPath)
		if err != nil {
			return nil, nil, err
		}
		if inf.IsDir() {
			f, err := fs.OpenFile(i.VirtualPath, os.O_RDONLY, 0, c.User.UID, c.User.GID)
			if err != nil {
				return nil, nil, err
			}
			defer f.Close()
			// Reads the directory and gets the information about the files.
			names, err := f.Readdirnames(-1)
			for _, n := range names {
				nMod := filepath.Join(i.VirtualPath, n)
				paths = append(paths, nMod)
				inf, err := fs.Stat(nMod)
				if err != nil {
					return nil, nil, err
				}
				files = append(files, inf)
			}
			if err != nil {
				return nil, nil, err
			}
		} else {
			p := filepath.Join(fs.String(), i.VirtualPath)
			if c.IsShare {
				inf, p, err = utils.ResolveSymlink(p)
				if err != nil {
					return nil, nil, err
				}
			}
			files = append(files, inf)
			paths = append(paths, i.VirtualPath)
		}
	}
	return files, paths, nil

}
func (i *File) listRecurs(c *Context, path string) (files []os.FileInfo, paths []string) {
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if c.IsShare {
				//get files from current user shares folder
				info, shrPath, _ := utils.ResolveSymlink(path)
				if info.IsDir() {
					infDir, err := os.OpenFile(shrPath, os.O_RDONLY, 0)
					if err != nil {
						return nil
					}
					//step to reduce problem in recursion
					names, err := infDir.Readdir(-1)
					if err != nil {
						return nil
					}
					for _, name := range names {
						if name.IsDir() {
							fr, pr := i.listRecurs(c, filepath.Join(path, name.Name()))
							files = append(files, fr...)
							paths = append(paths, pr...)
						} else {
							if c.FitFilter != nil && c.FitFilter(name.Name(), shrPath) ||
								c.FitFilter == nil {
								files = append(files, name)
								paths = append(paths, c.CutPath(filepath.Join(path, name.Name())))
							}
						}
					}

				}
				return nil

			} else {
				if c.FitFilter != nil && c.FitFilter(info.Name(), path) || c.FitFilter == nil {
					files = append(files, info)
					paths = append(paths, c.CutPath(path))
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
	for ind, f := range files {
		name := f.Name()

		//resolve share symlink
		if c.IsShare && !c.IsRecursive {
			f, _, _ = utils.ResolveSymlink(filepath.Join(i.Path, name))
		}

		if f.IsDir() {
			name += "/"
			dirCount++
		} else {
			fileCount++
		}
		fUrl = url.URL{Path: paths[ind]}
		fI := &File{
			Name:    f.Name(),
			Size:    f.Size(),
			ModTime: f.ModTime(),
			IsDir:   f.IsDir(),
			URL:     fUrl.String(),
		}

		_, fI.Type = utils.GetFileType(f.Name())

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
