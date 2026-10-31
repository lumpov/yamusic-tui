package tracker

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/config"
	"gopkg.in/yaml.v3"
)

func yamlVolume(t *testing.T) float64 {
	t.Helper()
	raw, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	var got config.Config
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	return got.Volume
}

func yamlShowLyrics(t *testing.T) bool {
	t.Helper()
	raw, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	var got config.Config
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	return got.ShowLyrics
}

func TestSetVolumeDoesNotWriteYamlUntilCmd(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	const want = 0.125
	before := yamlVolume(t)

	start := time.Now()
	cmd := m.SetVolume(want)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("SetVolume must not wait for disk")
	}
	if cmd == nil {
		t.Fatal("SetVolume must return Cmd")
	}
	if m.Volume() != want || config.Current.Volume != want {
		t.Fatalf("memory volume=%v current=%v", m.Volume(), config.Current.Volume)
	}
	if yamlVolume(t) != before {
		t.Fatal("yaml must not update until Cmd runs")
	}

	_ = cmd()
	if yamlVolume(t) != want {
		t.Fatalf("yaml volume=%v, want %v", yamlVolume(t), want)
	}
}

func TestSetLyricsDoesNotWriteYamlUntilCmd(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	want := !yamlShowLyrics(t)

	start := time.Now()
	cmd := m.SetLirycs(want)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("SetLirycs must not wait for disk")
	}
	if cmd == nil {
		t.Fatal("SetLirycs must return Cmd")
	}
	if m.showLyrics != want || config.Current.ShowLyrics != want {
		t.Fatal("memory lyrics flag not updated")
	}
	if yamlShowLyrics(t) == want {
		t.Fatal("yaml must not update until Cmd runs")
	}

	_ = cmd()
	if yamlShowLyrics(t) != want {
		t.Fatalf("yaml show-lyrics=%v, want %v", yamlShowLyrics(t), want)
	}
}

func diskConfig(t *testing.T) config.Config {
	t.Helper()
	raw, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	var got config.Config
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSetLyricsCmdKeepsTokenVolumeRewindCache(t *testing.T) {
	config.InitTest(t)
	config.Current.Token = "keep-lyrics-token"
	config.Current.Volume = 0.4
	config.Current.RewindDuration = 7
	config.Current.CacheTracks = config.CACHE_ALL
	config.Current.ShowLyrics = false
	m := NewForTest()

	cmd := m.SetLirycs(true)
	if cmd == nil {
		t.Fatal("SetLirycs must return Cmd")
	}
	_ = cmd()

	got := diskConfig(t)
	if got.Token != "keep-lyrics-token" {
		t.Fatal("token lost")
	}
	if got.Volume != 0.4 {
		t.Fatalf("volume=%v, want 0.4", got.Volume)
	}
	if got.RewindDuration != 7 {
		t.Fatalf("rewind=%v, want 7", got.RewindDuration)
	}
	if got.CacheTracks != config.CACHE_ALL {
		t.Fatalf("cache-tracks=%v, want all", got.CacheTracks)
	}
	if !got.ShowLyrics {
		t.Fatal("show-lyrics not saved")
	}
	if got.Controls == nil {
		t.Fatal("controls wiped")
	}
}

func TestSetLyricsEmptyTokenKeepsFileToken(t *testing.T) {
	config.InitTest(t)
	if err := os.WriteFile(config.Path(), []byte("token: keep-empty-save\nvolume: 0.4\nshow-lyrics: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config.Current.Token = ""
	config.Current.Volume = 0.4
	config.Current.ShowLyrics = false
	m := NewForTest()
	_ = m.SetLirycs(true)()

	got := diskConfig(t)
	if got.Token != "keep-empty-save" {
		t.Fatal("empty Current.Token replaced file token")
	}
	if !got.ShowLyrics {
		t.Fatal("show-lyrics not saved")
	}
	if got.Volume != 0.4 {
		t.Fatalf("volume=%v, want 0.4", got.Volume)
	}
}

func TestLyricsKeyTogglesMemory(t *testing.T) {
	config.InitTest(t)
	config.Current.ShowLyrics = false
	m := NewForTest()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("t must return Cmd")
	}
	if !m.showLyrics || !config.Current.ShowLyrics {
		t.Fatal("t must turn lyrics on")
	}
}
