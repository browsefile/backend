package fileutils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tempdir)

	folder := filepath.Join(tempdir, "folder")
	err = CopyDir("./testdata", folder)
	if err != nil {
		t.Error(err)
	}

	testdata := Dir(folder)

	if testdata.String() != folder {
		t.Error(errors.New("wrong path"))
	}

	err = testdata.Copy("/mountain", "/test")
	if err != nil {
		t.Error(err)
	}

	err = testdata.Rename("/test/everest.txt", "/test/everest2.txt")
	if err != nil {
		t.Error(err)
	}

	_, err = testdata.Stat("/test/everest2.txt")
	if err != nil {
		t.Error(err)
	}

	err = testdata.Mkdir("/test/qwe/rty/uio/pas/dfg/gh", 0700)
	if err != nil {
		t.Error(err)
	}

	err = testdata.RemoveAll("/test")
	if err != nil {
		t.Error(err)
	}
}
