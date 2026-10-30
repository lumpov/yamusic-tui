package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

type tester interface {
	Helper()
	TempDir() string
	Cleanup(func())
	Fatal(args ...any)
}

//go:embed testdata/config.yaml
var fixtureConfig []byte

var (
	testDir            string
	originalConfigPath string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	originalConfigPath = filepath.Join(home, ".config", DirName, _CONFIG_FILE)
}

func originalConfigBytes() []byte {
	if originalConfigPath != "" {
		if raw, err := os.ReadFile(originalConfigPath); err == nil && len(raw) > 0 {
			return raw
		}
	}
	return fixtureConfig
}

func InitTest(t tester) {
	t.Helper()
	raw := originalConfigBytes()
	dir := filepath.Join(t.TempDir(), ".config", DirName)
	if err := os.MkdirAll(dir, _DIR_PERM); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, _DIR_PERM); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, _CONFIG_FILE)
	if err := os.WriteFile(dst, raw, _CONFIG_PERM); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dst, _CONFIG_PERM); err != nil {
		t.Fatal(err)
	}
	testDir = dir
	t.Cleanup(func() { testDir = "" })
	Current = defaultConfig
	Current.CacheDir = filepath.Join(t.TempDir(), "cache")
}
