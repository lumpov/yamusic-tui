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

func TestSyncedLyricsIndexUsesStartTimestamp(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.lyrics = []api.LyricPair{
		{Timestamp: 10000, Line: "one"},
		{Timestamp: 20000, Line: "two"},
		{Timestamp: 30000, Line: "three"},
	}
	if idx := m.syncedLyricsIndex(0); idx != -1 {
		t.Fatalf("before first: idx=%d", idx)
	}
	if idx := m.syncedLyricsIndex(10000); idx != 0 {
		t.Fatalf("at first: idx=%d", idx)
	}
	if idx := m.syncedLyricsIndex(25000); idx != 1 {
		t.Fatalf("mid second: idx=%d", idx)
	}
	if idx := m.syncedLyricsIndex(90000); idx != 2 {
		t.Fatalf("after last: idx=%d", idx)
	}
}

func TestRenderLyricsKeepsLastLineAfterLastTimestamp(t *testing.T) {
	config.InitTest(t)
	data := bytes.Repeat([]byte("x"), 100)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()
	if _, err := buf.Seek(90, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	m := NewForTest()
	m.width = 80
	m.showLyrics = true
	m.player = &fakePlayer{playing: true}
	m.track = api.Track{DurationMs: 100000}
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	m.lyrics = []api.LyricPair{
		{Timestamp: 1000, Line: "one"},
		{Timestamp: 2000, Line: "final-line"},
	}
	got := m.renderLyrics()
	if !strings.Contains(got, "final-line") {
		t.Fatalf("last line must stay, got %q", got)
	}
}
