//go:build !darwin

package tracker

import (
	"bytes"
	"io"
	"testing"

	"github.com/dece2183/yamusic-tui/stream"
)

func TestNewPlaybackSourceRejectsInvalidMP3(t *testing.T) {
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})), 8)
	defer buf.Close()

	src, err := newPlaybackSource(buf)
	if err == nil {
		t.Fatal("invalid mp3 must return error")
	}
	if src != nil {
		t.Fatal("invalid mp3 must not become oto source")
	}
}

func TestNewPlaybackSourceDecodesValidMP3(t *testing.T) {
	data := testMP3Bytes(t)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	src, err := newPlaybackSource(buf)
	if err != nil {
		t.Fatal(err)
	}
	if src == nil {
		t.Fatal("decoder must be returned")
	}
}
