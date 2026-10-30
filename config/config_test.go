package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withHome(t *testing.T) string {
	t.Helper()
	testDir = ""
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
	broken := []byte("token: abc\nэто не yaml\n")
	path := configPath(home)
	if err := os.WriteFile(path, broken, 0755); err != nil {
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
	if Current.Token != "abc" {
		t.Fatalf("Token=%q, want abc", Current.Token)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _CONFIG_PERM {
		t.Fatalf("perm=%o, want %o", got, _CONFIG_PERM)
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

func TestInitialLoadChmodsExistingFile(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := configPath(home)
	if err := os.WriteFile(path, []byte("token: x\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := InitialLoad(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _CONFIG_PERM {
		t.Fatalf("perm=%o, want %o", got, _CONFIG_PERM)
	}
	if Current.Token != "x" {
		t.Fatalf("Token=%q, want x", Current.Token)
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

func TestSaveChmodsConfigDirOwnerOnly(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	Current = defaultConfig
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _DIR_PERM {
		t.Fatalf("dir perm=%o, want %o", got, _DIR_PERM)
	}
}

func TestSaveDoesNotReplaceExistingTokenWithEmpty(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := configPath(home)
	if err := os.WriteFile(path, []byte("token: keep-me\nvolume: 0.2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	Current = defaultConfig
	if Current.Token != "" {
		t.Fatal("default token must be empty")
	}
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if tok := tokenFromYAML(got); tok != "keep-me" {
		t.Fatalf("token=%q, want keep-me", tok)
	}
}

func TestInitTestPathIsNotOriginalConfig(t *testing.T) {
	InitTest(t)
	path := Path()
	if path == "" {
		t.Fatal("Path empty")
	}
	if originalConfigPath != "" && path == originalConfigPath {
		t.Fatal("InitTest Path must not be product config.yaml")
	}
}

func TestInitTestSaveDoesNotTouchProductConfig(t *testing.T) {
	live := originalConfigPath
	var before []byte
	var mtime os.FileInfo
	if live != "" {
		if b, err := os.ReadFile(live); err == nil {
			before = b
			mtime, _ = os.Stat(live)
		}
	}
	InitTest(t)
	Current.Token = ""
	Current.Volume = 0.11
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	if live == "" {
		return
	}
	after, err := os.ReadFile(live)
	if len(before) == 0 {
		if err == nil {
			t.Fatal("product config.yaml created by InitTest Save")
		}
		return
	}
	if err != nil {
		t.Fatal("product config.yaml disappeared")
	}
	st, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	if mtime != nil && !st.ModTime().Equal(mtime.ModTime()) {
		t.Fatal("product config.yaml mtime changed")
	}
	if string(before) != string(after) {
		t.Fatal("product config.yaml content changed")
	}
	got, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if want := tokenFromYAML(before); want != "" && tokenFromYAML(got) != want {
		t.Fatal("test copy lost token from original")
	}
}

func TestInitTestCopiesOriginalEachCall(t *testing.T) {
	InitTest(t)
	Current.Token = "spoiled"
	Current.Volume = 0.01
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	InitTest(t)
	got, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	orig := originalConfigBytes()
	if tokenFromYAML(got) == "spoiled" && tokenFromYAML(orig) != "spoiled" {
		t.Fatal("InitTest reused spoiled test config")
	}
	if tokenFromYAML(got) != tokenFromYAML(orig) {
		t.Fatal("InitTest copy token must match original")
	}
}

func TestInitTestUsesFixtureWhenProductMissing(t *testing.T) {
	prev := originalConfigPath
	originalConfigPath = filepath.Join(t.TempDir(), "nope.yaml")
	t.Cleanup(func() { originalConfigPath = prev })
	InitTest(t)
	got, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if tokenFromYAML(got) != "test-token" {
		t.Fatal("missing product must copy testdata fixture")
	}
}
