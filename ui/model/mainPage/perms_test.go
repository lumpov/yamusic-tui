package mainpage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
)

func TestTempMediaDirAndFilesAreOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes")
	}
	config.InitTest()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	dir := filepath.Join(tmp, config.DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	cover := m.coverFilePath(&api.Track{Id: "1"})
	if cover == "" {
		t.Fatal("cover path empty")
	}
	dirStat, err := os.Stat(filepath.Dir(cover))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirStat.Mode().Perm(); got != 0700 {
		t.Fatalf("temp dir perm=%o, want 0700", got)
	}

	f, err := openOwnerFile(cover, os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	stat, err := os.Stat(cover)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != 0600 {
		t.Fatalf("cover perm=%o, want 0600", got)
	}
}
