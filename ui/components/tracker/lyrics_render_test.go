package tracker

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

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

func TestPositionUsesPlayerCurrentTime(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.track = api.Track{DurationMs: 60000}
	m.playheadOrigin = 10 * time.Second
	m.player = &fakePlayer{hasTime: true, currentTime: 2 * time.Second}
	if got := m.Position(); got != 12*time.Second {
		t.Fatalf("got %s, want 12s", got)
	}
}

func TestPositionFallsBackToByteProgress(t *testing.T) {
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
	m.player = &fakePlayer{}
	if got := m.Position().Milliseconds(); got != 80000 {
		t.Fatalf("got %d, want 80000", got)
	}
}

func TestPositionClampsToDuration(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.track = api.Track{DurationMs: 1000}
	m.playheadOrigin = 900 * time.Millisecond
	m.player = &fakePlayer{hasTime: true, currentTime: 2 * time.Second}
	if got := m.Position(); got != time.Second {
		t.Fatalf("got %s, want 1s", got)
	}
}

func TestRenderLyricsFollowsPlayerClockNotBytes(t *testing.T) {
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
	m.track = api.Track{DurationMs: 100000}
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	m.playheadOrigin = 0
	m.player = &fakePlayer{playing: true, hasTime: true, currentTime: 1500 * time.Millisecond}
	m.lyrics = []api.LyricPair{
		{Timestamp: 1000, Line: "clock-line"},
		{Timestamp: 20000, Line: "feed-ahead-line"},
	}
	if idx := m.syncedLyricsIndex(int(m.Position().Milliseconds())); idx != 0 {
		t.Fatalf("idx=%d, want 0; byte progress 90%% would pick the later line", idx)
	}
	got := m.renderLyrics()
	if !strings.Contains(got, "clock-line") {
		t.Fatalf("want clock line, got %q", got)
	}
}

func TestRewindUsesPlayheadNotFeed(t *testing.T) {
	config.InitTest(t)
	data := bytes.Repeat([]byte("x"), 60000)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), 60000)
	defer buf.Close()
	if _, err := buf.Seek(40000, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	fp := &fakePlayer{playing: true, hasTime: true, currentTime: 10 * time.Second}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000}
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	m.Rewind(5 * time.Second)
	if fp.offset != 15000 {
		t.Fatalf("offset=%d, want 15000 (playhead 10s + 5s)", fp.offset)
	}
	if m.playheadOrigin != 15*time.Second {
		t.Fatalf("origin=%s, want 15s", m.playheadOrigin)
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
