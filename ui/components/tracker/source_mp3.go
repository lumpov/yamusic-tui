//go:build !darwin

package tracker

import (
	"io"

	mp3 "github.com/dece2183/go-stream-mp3"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/stream"
)

func newPlaybackSource(reader *stream.BufferedStream) (io.ReadSeeker, error) {
	decoder, err := mp3.NewDecoder(reader)
	if err != nil {
		log.Print(log.LVL_ERROR, "failed to create mp3 decoder: %s", err)
		return nil, err
	}
	return decoder, nil
}
