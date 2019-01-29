package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tempdir)

	file := filepath.Join(tempdir, "file.txt")

	err = CopyFile("./testdata/file.txt", file)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(file)
	if err != nil && err == os.ErrNotExist {
		t.Error(err)
	}
}

func TestCopyDir(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tempdir)

	folder := filepath.Join(tempdir, "folder/folder")

	err = CopyDir("./testdata/mountain", folder)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(folder)
	if err != nil && err == os.ErrNotExist {
		t.Error(err)
	}

	_, err = os.Stat(filepath.Join(folder, "everest.txt"))
	if err != nil && err == os.ErrNotExist {
		t.Error(err)
	}
}
