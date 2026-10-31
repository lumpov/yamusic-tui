package tracker

import (
	"os"
	"testing"
	"time"

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
