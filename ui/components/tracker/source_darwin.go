//go:build darwin

package tracker

import (
	"io"

	"github.com/dece2183/yamusic-tui/stream"
)

func newPlaybackSource(reader *stream.BufferedStream) (io.ReadSeeker, error) {
	return reader, nil
}
