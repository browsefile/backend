package config

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)

	cfg.WriteConfig()

	_, err := os.Stat(cfg.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(cfg.GetUserHomePath("admin")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(cfg.GetUserSharesPath("admin")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(cfg.GetUserPreviewPath("admin")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(cfg.GetUserSharexPath("admin")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(cfg.GetDavPath("admin")); err != nil {
		t.Fatal(err)
	}
	if config == nil {
		t.Fatal("global config empty")
	}
}
func TestNewModConfig(t *testing.T) {
	cfg := TContext{}
	cfg.Init()
	defer cfg.Clean(t)
	cfg.WriteConfig()
	cfg2 := cfg.CopyConfig()
	cfg2.Http.Port = 80
	cfg.UpdateConfig(cfg2)
	cfg.WriteConfig()
	cfg3 := GlobalConfig{Path: cfg.Path, FilesPath: cfg.FilesPath,}
	cfg3.ReadConfigFile()
	if cfg3.Http.Port != 80 {
		t.Fatal("config update fail")
	}
	arr, _ := cfg.GetKeyBytes()
	if len(arr) == 0 {
		t.Fatal("initial key must be generated")
	}

}

func TestFirstRunPreviewGen(t *testing.T) {
	cfg := TContext{}
	cfg.InitWithUsers(t)
	cfg.PreviewConf.FirstRun = true
	cfg.WriteConfig()
	cfg.ReadConfigFile()
}
