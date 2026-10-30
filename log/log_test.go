package log

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestStartCreatesOwnerOnlyLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes")
	}
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	config.InitTest(t)

	Start()
	defer Stop()

	path := Location()
	if path == "" {
		t.Fatal("log path empty")
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := stat.Mode().Perm(); got != _FILE_PERM {
		t.Fatalf("log perm=%o, want %o", got, _FILE_PERM)
	}
	dirStat, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirStat.Mode().Perm(); got != _DIR_PERM {
		t.Fatalf("log dir perm=%o, want %o", got, _DIR_PERM)
	}
}

func TestStartDeletesPreviousLog(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	config.InitTest(t)

	Start()
	path := Location()
	if path == "" {
		t.Fatal("log path empty")
	}
	Print(LVL_INFO, "old session")
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(old) == 0 {
		t.Fatal("first session must write")
	}

	Start()
	defer Stop()
	if Location() != path {
		t.Fatalf("path changed: %s", Location())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("previous log must be deleted, got %q", got)
	}
}

func TestDumpWritesBlock(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	config.InitTest(t)

	Start()
	defer Stop()
	Dump("playbacktrace reason=stop n=1\n+0.000 rewind\n")
	got, err := os.ReadFile(Location())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "playbacktrace reason=stop n=1\n+0.000 rewind\n" {
		t.Fatalf("got=%q", got)
	}
}
