package tracker

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/stream"
)

func testMP3Bytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/mpeg2-head.mp3")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

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
	opens  int
}

func (b *fakeBackend) Open(src *readWrapper) audioPlayer {
	b.opens++
	if b.player == nil {
		b.player = &fakePlayer{}
	}
	return b.player
}

func TestStartTrackDoesNotOpenOnDecoderError(t *testing.T) {
	config.InitTest(t)
	prev := playbackSource
	playbackSource = func(*stream.BufferedStream) (io.ReadSeeker, error) {
		return nil, io.ErrUnexpectedEOF
	}
	defer func() { playbackSource = prev }()

	m := NewForTest()
	b := m.audio.(*fakeBackend)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})), 8)
	defer buf.Close()

	m.StartTrack(&api.Track{Id: "1", Title: "t", DurationMs: 1000, Available: true}, buf, nil)

	if b.opens != 0 {
		t.Fatal("Open must not run when decoder fails")
	}
	if m.player != nil {
		t.Fatal("player must stay nil")
	}
	if !m.showError || m.errorText != "mp3 decoder" {
		t.Fatalf("error=%q", m.errorText)
	}
}

func TestPlayPauseStopUseAudioPlayer(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	fp := m.audio.(*fakeBackend).player

	data := testMP3Bytes(t)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
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
	config.InitTest(t)
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
	config.InitTest(t)
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
	config.InitTest(t)
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
	config.InitTest(t)
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
	config.InitTest(t)
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
	config.InitTest(t)
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
	config.InitTest(t)
	m := NewForTest()
	m.Update(VolumeTo(0.3))
	if m.volume != 0.3 {
		t.Fatalf("volume=%f, want 0.3", m.volume)
	}
}

func TestNeedsVolumeFadeFalseWhenAtTarget(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.player = &fakePlayer{playing: true, volume: 0.5}
	m.volume = 0.5
	m.paused = false
	m.volumeIncremet = 0.25
	if m.needsVolumeFade() {
		t.Fatal("at target must not fade")
	}
}

func TestNeedsVolumeFadeTrueWhilePausing(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	m.player = &fakePlayer{playing: true, volume: 0.5}
	m.volume = 0.5
	m.paused = true
	m.volumeIncremet = 0.25
	if !m.needsVolumeFade() {
		t.Fatal("pause fade-out must tick")
	}
}

func TestVolumeFadeTickMsgPausesPlayer(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	fp := m.audio.(*fakeBackend).player
	data := testMP3Bytes(t)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
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
