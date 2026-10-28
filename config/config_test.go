package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func configPath(home string) string {
	return filepath.Join(home, ".config", DirName, _CONFIG_FILE)
}

func TestSaveCreatesOwnerOnlyFile(t *testing.T) {
	home := withHome(t)
	Current = defaultConfig
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(configPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _CONFIG_PERM {
		t.Fatalf("perm=%o, want %o", got, _CONFIG_PERM)
	}
}

func TestInitialLoadMissingFileCreatesDefault(t *testing.T) {
	home := withHome(t)
	if err := InitialLoad(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath(home)); err != nil {
		t.Fatal(err)
	}
	if Current.Controls == nil {
		t.Fatal("Controls must be filled from defaults")
	}
}

func TestInitialLoadInvalidYAMLDoesNotOverwrite(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	broken := []byte("token: [\nnot yaml")
	path := configPath(home)
	if err := os.WriteFile(path, broken, 0600); err != nil {
		t.Fatal(err)
	}
	err := InitialLoad()
	if err == nil {
		t.Fatal("expected YAML error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(broken) {
		t.Fatalf("file overwritten: %q", got)
	}
}

func TestInitialLoadTrimsOldTokenFile(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte("abc\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitialLoad(); err != nil {
		t.Fatal(err)
	}
	if Current.Token != "abc" {
		t.Fatalf("Token=%q, want abc", Current.Token)
	}
}

func TestSaveChmodsExistingWorldReadableFile(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := configPath(home)
	if err := os.WriteFile(path, []byte("token: x\n"), 0755); err != nil {
		t.Fatal(err)
	}
	Current = defaultConfig
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _CONFIG_PERM {
		t.Fatalf("perm=%o, want %o", got, _CONFIG_PERM)
	}
}
