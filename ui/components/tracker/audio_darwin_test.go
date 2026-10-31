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

func TestDarwinPlayerCurrentTimeBeforeQueue(t *testing.T) {
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(nil)), 0)
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf}
	p := newDarwinPlayer(w)
	defer p.Close()

	got, ok := p.CurrentTime()
	if !ok {
		t.Fatal("CurrentTime must report playhead before queue start")
	}
	if got != 0 {
		t.Fatalf("got %s, want 0", got)
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
	if err := w.NewReader(buf); err != nil {
		t.Fatal(err)
	}
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

func TestDarwinPlayerSeekAfterEOFRestartsFeed(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 256)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf, source: buf, emitNextOnEOF: false}
	p := newDarwinPlayer(w)
	defer p.Close()

	p.Play()
	time.Sleep(30 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := p.Seek(0, io.SeekStart)
		p.Play()
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Seek blocked")
	}

	time.Sleep(30 * time.Millisecond)
	p.mu.Lock()
	feeding := p.feeding
	p.mu.Unlock()
	if !feeding {
		t.Fatal("Play after Seek must restart feed")
	}
}

func TestDarwinSeekDuringPauseDoesNotBlock(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 256)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf, source: buf, emitNextOnEOF: false}
	p := newDarwinPlayer(w)
	defer p.Close()

	p.Play()
	time.Sleep(30 * time.Millisecond)
	p.Pause()

	done := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < 20; i++ {
			_, err = p.Seek(int64(i*16), io.SeekStart)
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Seek blocked during Pause")
	}
}

func TestDarwinPlayerCloseWaitsForFeedThenDestroys(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 256)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{trackBuffer: buf, source: buf, emitNextOnEOF: false}
	p := newDarwinPlayer(w)
	p.Play()
	time.Sleep(30 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- p.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close blocked")
	}

	p.mu.Lock()
	feeding := p.feeding
	c := p.c
	p.mu.Unlock()
	if feeding {
		t.Fatal("Close must stop feed before destroy")
	}
	if c != nil {
		t.Fatal("Close must clear AQPlayer after feed exits")
	}
}
