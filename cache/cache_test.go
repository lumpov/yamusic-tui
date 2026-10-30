package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestWriteCreatesOwnerOnlyCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes")
	}
	config.InitTest()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	config.Current.CacheDir = dir

	file, err := Write("1")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	stat, err := os.Stat(filepath.Join(dir, "1.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _FILE_PERM {
		t.Fatalf("cache file perm=%o, want %o", got, _FILE_PERM)
	}
	dirStat, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirStat.Mode().Perm(); got != _DIR_PERM {
		t.Fatalf("cache dir perm=%o, want %o", got, _DIR_PERM)
	}
}
