package utils

import (
	"bytes"
	"errors"
	"github.com/browsefile/backend/src/cnst"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from source to dest and returns
// an error if any.
func CopyFile(source string, dest string, uid, gid int) error {
	// Open the source file.
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	// Makes the directory needed to create the dst
	// file.
	err = os.MkdirAll(filepath.Dir(dest), cnst.PERM_DEFAULT)
	if err != nil {
		return err
	}
	// Create the destination file.
	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the contents of the file.
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Copy the mode if the user can't
	// open the file.
	info, err := os.Stat(source)
	if err != nil {
		err = os.Chmod(dest, info.Mode())
		ModPermission(uid, gid, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

// CopyDir copies a directory from source to dest and all
// of its sub-directories. It doesn't stop if it finds an error
// during the copy. Returns an error if any.
func CopyDir(source string, dest string, uid, gid int) error {
	// Get properties of source.
	srcinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Create the destination directory.
	err = os.MkdirAll(dest, srcinfo.Mode())
	if err != nil {
		return err
	}

	dir, _ := os.Open(source)
	obs, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	var errStr bytes.Buffer
	for _, obj := range obs {
		fsource := source + "/" + obj.Name()
		fdest := dest + "/" + obj.Name()

		if obj.IsDir() {
			// Create sub-directories, recursively.
			err = CopyDir(fsource, fdest, uid, gid)
			if err != nil {
				errStr.WriteString(err.Error())
				errStr.WriteRune('\n')
			}
		} else {
			// Perform the file copy.
			err = CopyFile(fsource, fdest, uid, gid)
			if err != nil {
				errStr.WriteString(err.Error())
				errStr.WriteRune('\n')
			}
		}
	}

	if errStr.Len() > 0 {
		return errors.New(errStr.String())
	}

	return nil
}
