//go:build !darwin

package tracker

import (
	"time"

	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/ui/model"
	"github.com/ebitengine/oto/v3"
)

const otoBytesPerSec = 44100 * 2 * 2

var _ audioPlayer = (*otoPlayer)(nil)

type otoBackend struct {
	ctx *oto.Context
}

type otoPlayer struct {
	p         *oto.Player
	src       *readWrapper
	originPCM int64
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
	return &otoPlayer{p: b.ctx.NewPlayer(src), src: src}
}

func (p *otoPlayer) Play()               { p.p.Play() }
func (p *otoPlayer) Pause()              { p.p.Pause() }
func (p *otoPlayer) Close() error        { return p.p.Close() }
func (p *otoPlayer) SetVolume(v float64) { p.p.SetVolume(v) }
func (p *otoPlayer) Volume() float64     { return p.p.Volume() }
func (p *otoPlayer) IsPlaying() bool     { return p.p.IsPlaying() }

func (p *otoPlayer) Seek(offset int64, whence int) (int64, error) {
	pos, err := p.p.Seek(offset, whence)
	if err == nil {
		p.originPCM = pos
	}
	return pos, err
}

func (p *otoPlayer) CurrentTime() (time.Duration, bool) {
	if p.p == nil || p.src == nil {
		return 0, false
	}
	played := p.src.Offset() - int64(p.p.BufferedSize())
	rel := played - p.originPCM
	if rel < 0 {
		rel = 0
	}
	return time.Duration(rel) * time.Second / otoBytesPerSec, true
}
