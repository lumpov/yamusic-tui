//go:build darwin

package tracker

import (
	"bytes"
	"io"
	"testing"

	"github.com/dece2183/yamusic-tui/stream"
)

func TestDarwinPlaybackSourceIsRawStream(t *testing.T) {
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader([]byte{1, 2, 3})), 3)
	defer buf.Close()

	src, err := newPlaybackSource(buf)
	if err != nil {
		t.Fatal(err)
	}
	if src != buf {
		t.Fatal("darwin must pass the stream to AudioQueue, not an mp3 decoder")
	}
}
