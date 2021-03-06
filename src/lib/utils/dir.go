package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// A Dir uses the native file system restricted to a specific directory tree.
// Originally from ttps://github.com/golang/net/blob/master/webdav/file.go
// An empty Dir is treated as ".".
type Dir string

func (d Dir) resolve(name string) string {
	// This implementation is based on Dir.Open's code in the standard net/web package.
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return ""
	}

	dir := string(d)
	if dir == "" {
		dir = "."
	}

	return filepath.Join(dir, filepath.FromSlash(SlashClean(name)))
}

// Mkdir implements os.Mkdir in this directory context.
func (d Dir) Mkdir(name string, perm os.FileMode, uid, gid int) error {
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}
	err := os.MkdirAll(name, perm)
	if err == nil {
		err = ModPermission(uid, gid, name)
	}
	return err
}

// OpenFile implements os.OpenFile in this directory context.
func (d Dir) OpenFile(name string, flag int, perm os.FileMode, uid, gid int) (*os.File, error) {
	if name = d.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	err = ModPermission(uid, gid, name)
	return f, err
}

// RemoveAll implements os.RemoveAll in this directory context.
func (d Dir) RemoveAll(name string) error {
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}

	if name == filepath.Clean(string(d)) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}
	return os.RemoveAll(name)
}

// Rename implements os.Rename in this directory context.
func (d Dir) Rename(oldName, newName string) error {
	if oldName = d.resolve(oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = d.resolve(newName); newName == "" {
		return os.ErrNotExist
	}
	if root := filepath.Clean(string(d)); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}
	return os.Rename(oldName, newName)
}

// Stat implements os.Stat in this directory context.
func (d Dir) Stat(name string) (os.FileInfo, error) {
	if name = d.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}

	return os.Stat(name)
}

// Copy copies a file or directory from src to dst. If it is
// a directory, all of the files and sub-directories will be copied.
func (d Dir) Copy(src, dst string, uid, gid int) error {
	if src = d.resolve(src); src == "" {
		return os.ErrNotExist
	}

	if dst = d.resolve(dst); dst == "" {
		return os.ErrNotExist
	}

	if root := filepath.Clean(string(d)); root == src || root == dst {
		// Prohibit copying from or to the virtual root directory.
		return os.ErrInvalid
	}

	if dst == src {
		return os.ErrInvalid
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return CopyDir(src, dst, uid, gid)
	}

	return CopyFile(src, dst, uid, gid)
}

func (d Dir) String() string {
	return string(d)
}
