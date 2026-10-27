package tracker

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/stream"
)

type fakePlayer struct {
	playing bool
	volume  float64
	offset  int64
	closed  bool
	plays   int
	pauses  int
}

func (p *fakePlayer) Play()               { p.playing = true; p.plays++ }
func (p *fakePlayer) Pause()              { p.playing = false; p.pauses++ }
func (p *fakePlayer) Close() error        { p.closed = true; p.playing = false; return nil }
func (p *fakePlayer) SetVolume(v float64) { p.volume = v }
func (p *fakePlayer) Volume() float64     { return p.volume }
func (p *fakePlayer) Seek(offset int64, whence int) (int64, error) {
	p.offset = offset
	return offset, nil
}
func (p *fakePlayer) IsPlaying() bool { return p.playing }

type fakeBackend struct {
	player *fakePlayer
}

func (b *fakeBackend) Open(src *readWrapper) audioPlayer {
	if b.player == nil {
		b.player = &fakePlayer{}
	}
	return b.player
}

func TestPlayPauseStopUseAudioPlayer(t *testing.T) {
	config.InitTest()
	m := NewForTest()
	fp := m.audio.(*fakeBackend).player

	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})), 8)
	defer buf.Close()

	track := &api.Track{Id: "1", Title: "t", DurationMs: 1000, Available: true}
	m.StartTrack(track, buf, nil)

	if fp.plays == 0 || !fp.playing {
		t.Fatal("StartTrack must call Play")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && fp.volume == 0 {
		time.Sleep(20 * time.Millisecond)
	}

	m.Pause()
	if !m.paused {
		t.Fatal("Pause must set tracker paused flag")
	}

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) && fp.pauses == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if fp.pauses == 0 {
		t.Fatal("volume fade must call Pause on the player")
	}

	m.Stop()
	if !fp.closed {
		t.Fatal("Stop must close the player")
	}
}

func TestRewindSeeksCompressedOffset(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.track = api.Track{DurationMs: 1000}
	m.trackWrapper = &readWrapper{
		trackBuffer: stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 1000))), 1000),
	}
	defer m.trackWrapper.trackBuffer.Close()

	m.Rewind(100 * time.Millisecond)
	if fp.offset == 0 {
		t.Fatal("Rewind must seek the player")
	}
}
