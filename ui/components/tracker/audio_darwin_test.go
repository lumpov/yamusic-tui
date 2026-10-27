//go:build darwin

package tracker

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/stream"
)

func TestDarwinBackendType(t *testing.T) {
	b := newAudioBackend()
	if _, ok := b.(darwinBackend); !ok {
		t.Fatal("macOS must use darwinBackend")
	}
}

func TestDarwinPlayerOpenCloseWithoutPlay(t *testing.T) {
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(nil)), 0)
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf}
	p := newDarwinPlayer(w)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDarwinPlayerPlayPauseVolume(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 64)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf}
	w.NewReader(buf)
	p := newDarwinPlayer(w)
	defer p.Close()

	p.SetVolume(0.25)
	if p.Volume() != 0.25 {
		t.Fatalf("volume=%f", p.Volume())
	}

	p.Play()
	if !p.IsPlaying() {
		t.Fatal("Play must report IsPlaying")
	}
	p.Pause()
	time.Sleep(30 * time.Millisecond)
	if p.IsPlaying() {
		t.Fatal("Pause must clear IsPlaying")
	}
}
