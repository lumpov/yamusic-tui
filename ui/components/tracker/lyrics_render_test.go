package tracker

import (
	"strings"
	"testing"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
)

func TestRenderLyricsEmpty(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.width = 80
	m.showLyrics = true
	m.player = &fakePlayer{playing: true}

	got := m.renderLyrics()
	if !strings.Contains(got, "No lyrics for this track") {
		t.Fatalf("empty lyrics: %q", got)
	}
}

func TestRenderLyricsWithoutTimestamps(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.width = 80
	m.showLyrics = true
	m.player = &fakePlayer{playing: true}
	m.lyrics = []api.LyricPair{
		{Timestamp: 0, Line: "Coming out the alleyways of Illtown"},
		{Timestamp: 0, Line: "Producer extraordinaire Kaygee"},
	}

	got := m.renderLyrics()
	if !strings.Contains(got, "No lyrics timestamps for this track") {
		t.Fatalf("want timestamps message, got %q", got)
	}
	if strings.Contains(got, "Coming out") || strings.Contains(got, "Kaygee") {
		t.Fatalf("must not show unsynced lyrics: %q", got)
	}
}
