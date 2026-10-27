//go:build !darwin

package tracker

import (
	"time"

	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/ui/model"
	"github.com/ebitengine/oto/v3"
)

var _ audioPlayer = (*oto.Player)(nil)

type otoBackend struct {
	ctx *oto.Context
}

func newAudioBackend() audioBackend {
	op := &oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		BufferSize:   time.Millisecond * time.Duration(config.Current.BufferSize),
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		log.Print(log.LVL_PANIC, "failed to create player context: %s", err)
		model.PrettyExit(err, 12)
	}
	<-readyChan

	return &otoBackend{ctx: ctx}
}

func (b *otoBackend) Open(src *readWrapper) audioPlayer {
	src.emitNextOnEOF = true
	return b.ctx.NewPlayer(src)
}
