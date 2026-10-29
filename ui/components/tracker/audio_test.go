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

func TestRewindStartStaysAtZero(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000}
	m.trackWrapper = &readWrapper{
		trackBuffer: stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 60000))), 60000),
	}
	defer m.trackWrapper.trackBuffer.Close()

	m.Rewind(-5 * time.Second)
	if fp.offset != 0 {
		t.Fatalf("start offset=%d, want 0", fp.offset)
	}
}

func TestRewindEndKeepsRewindStep(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000}
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 60000))), 60000)
	defer buf.Close()
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	_, _ = buf.Seek(59000, io.SeekStart)

	m.Rewind(5 * time.Second)
	if fp.offset >= 60000 {
		t.Fatalf("must not seek to end, offset=%d", fp.offset)
	}
	if fp.offset != 55000 {
		t.Fatalf("offset=%d, want 55000", fp.offset)
	}
}

func TestRewindPausesUntilSettle(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000}
	m.trackWrapper = &readWrapper{
		trackBuffer: stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 60000))), 60000),
	}
	defer m.trackWrapper.trackBuffer.Close()

	m.Rewind(5 * time.Second)
	if !m.paused {
		t.Fatal("rewind burst must pause")
	}
	if fp.pauses == 0 {
		t.Fatal("player must Pause during rewind")
	}
	plays := fp.plays
	m.Rewind(5 * time.Second)
	if fp.plays != plays {
		t.Fatal("second rewind must not Play")
	}
	m.finishRewindBurst()
	if fp.plays <= plays {
		t.Fatal("settle must Play")
	}
	if m.paused {
		t.Fatal("settle must clear paused")
	}
}

func TestPlayPauseControlTogglesPlayback(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.trackWrapper = &readWrapper{
		trackBuffer: stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 8))), 8),
	}
	defer m.trackWrapper.trackBuffer.Close()

	m.Update(PLAYPAUSE)
	if !m.paused {
		t.Fatal("PLAYPAUSE while playing must pause")
	}

	fp.playing = false
	m.Update(PLAYPAUSE)
	if m.paused {
		t.Fatal("PLAYPAUSE while paused must play")
	}
}

func TestSeekByUpdateRewinds(t *testing.T) {
	config.InitTest()
	fp := &fakePlayer{playing: true}
	m := NewForTest()
	m.player = fp
	m.paused = false
	m.rewindAmount = 5 * time.Second
	m.track = api.Track{DurationMs: 60000}
	m.trackWrapper = &readWrapper{
		trackBuffer: stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 60000))), 60000),
	}
	defer m.trackWrapper.trackBuffer.Close()

	_, cmd := m.Update(SeekBy(5 * time.Second))
	if cmd == nil {
		t.Fatal("SeekBy must return settle command")
	}
	if fp.offset == 0 {
		t.Fatal("SeekBy must seek")
	}
}

func TestVolumeToUpdateSetsVolume(t *testing.T) {
	config.InitTest()
	m := NewForTest()
	m.Update(VolumeTo(0.3))
	if m.volume != 0.3 {
		t.Fatalf("volume=%f, want 0.3", m.volume)
	}
}

func TestVolumeFadeTickMsgPausesPlayer(t *testing.T) {
	config.InitTest()
	m := NewForTest()
	fp := m.audio.(*fakeBackend).player
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})), 8)
	defer buf.Close()
	m.StartTrack(&api.Track{Id: "1", Title: "t", DurationMs: 1000, Available: true}, buf, nil)
	fp.volume = m.volume
	m.Pause()

	for i := 0; i < 8 && fp.pauses == 0; i++ {
		m.Update(volumeFadeTickMsg{})
	}
	if fp.pauses == 0 {
		t.Fatal("volumeFadeTickMsg must Pause the player after fade-out")
	}
}
