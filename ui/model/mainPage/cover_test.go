package mainpage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestCoverFilePathIsSingleSharedFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)

	m := &Model{}
	got := m.coverFilePath()
	want := filepath.Join(tmp, config.DirName, "cover.jpg")
	if got != want {
		t.Fatalf("coverFilePath: got %q want %q", got, want)
	}
}

func TestRemoveCoverFilesDeletesJpgsOnly(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)

	dir := filepath.Join(tmp, config.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "59263985.jpg")
	current := filepath.Join(dir, "cover.jpg")
	meta := filepath.Join(dir, "metadata.mp3")
	logf := filepath.Join(dir, "log")
	for _, p := range []string{stale, current, meta, logf} {
		if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	m := &Model{}
	m.removeCoverFiles()

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale id jpg still present: %v", err)
	}
	if _, err := os.Stat(current); !os.IsNotExist(err) {
		t.Fatalf("cover.jpg still present: %v", err)
	}
	if _, err := os.Stat(meta); err != nil {
		t.Fatalf("metadata.mp3 removed: %v", err)
	}
	if _, err := os.Stat(logf); err != nil {
		t.Fatalf("log removed: %v", err)
	}
}
