package tracker

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/stream"
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
	if strings.Contains(got, "No lyrics timestamps for this track") {
		t.Fatal("plain lyrics must be shown")
	}
	if !strings.Contains(got, "Coming out") {
		t.Fatalf("want first line, got %q", got)
	}
	if !strings.Contains(got, "Kaygee") {
		t.Fatalf("want second line, got %q", got)
	}
}

func TestUnsyncedLyricsIndexFollowsProgress(t *testing.T) {
	config.InitTest(t)
	data := bytes.Repeat([]byte("x"), 100)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()
	if _, err := buf.Seek(80, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	m := NewForTest()
	m.track = api.Track{DurationMs: 100000}
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	m.lyrics = []api.LyricPair{
		{Line: "one"},
		{Line: "two"},
		{Line: "three"},
		{Line: "four"},
		{Line: "five"},
	}

	idx := m.unsyncedLyricsIndex()
	if idx != 4 {
		t.Fatalf("idx=%d, want 4", idx)
	}
}
