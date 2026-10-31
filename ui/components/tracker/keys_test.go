package tracker

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/stream"
)

func TestVolumeKeysAreBrackets(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.volume = 0.5
	m.volumeIncremet = 0.1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if cmd == nil {
		t.Fatal("] must change volume")
	}
	if m.volume <= 0.5 {
		t.Fatalf("] must raise volume, got %f", m.volume)
	}

	before := m.volume
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if cmd == nil {
		t.Fatal("[ must change volume")
	}
	if m.volume >= before {
		t.Fatalf("[ must lower volume, got %f", m.volume)
	}
}

func TestPlusDoesNotChangeVolume(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.volume = 0.5
	before := m.volume
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if m.volume != before {
		t.Fatal("+ must not change volume")
	}
}

func TestShiftRightRewindsCurrentTrack(t *testing.T) {
	config.InitTest(t)
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000, Available: true}
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 60000))), 60000)
	defer buf.Close()
	m.trackWrapper = &readWrapper{trackBuffer: buf}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	if cmd == nil {
		t.Fatal("shift+right must rewind")
	}
	if fp.offset == 0 {
		t.Fatal("shift+right must seek inside the track")
	}
	if fp.offset >= 60000 {
		t.Fatal("rewind must not seek to end")
	}

	before := fp.offset
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd == nil {
		t.Fatal("right must emit next")
	}
	if fp.offset != before {
		t.Fatal("right must not rewind")
	}
}

type stallSrc struct {
	unblock chan struct{}
}

func (s *stallSrc) Read([]byte) (int, error) {
	<-s.unblock
	return 0, io.EOF
}

func (s *stallSrc) Close() error { return nil }

func TestCacheKeyDoesNotWaitBufferAll(t *testing.T) {
	config.InitTest(t)
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	src := &stallSrc{unblock: make(chan struct{})}
	buf := stream.NewBufferedStream(src, 1<<20)
	defer func() {
		close(src.unblock)
		buf.Close()
	}()
	m.trackWrapper.trackBuffer = buf

	start := time.Now()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("cache key must not wait for BufferAll")
	}
	if cmd == nil {
		t.Fatal("S must emit CACHE_TRACK")
	}
}
