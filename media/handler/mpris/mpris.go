//go:build linux && !nomedia

package mpris

import (
	"sync"
	"time"

	"github.com/dece2183/yamusic-tui/media/handler"
	"github.com/quarckster/go-mpris-server/pkg/events"
	"github.com/quarckster/go-mpris-server/pkg/server"
	"github.com/quarckster/go-mpris-server/pkg/types"
)

type MprisHandler struct {
	server      *server.Server
	evHandler   *events.EventHandler
	name        string
	description string
	mu          sync.Mutex
	closed      bool
	msgChan     chan handler.Message
	ansChan     chan any
}

func NewHandler(name, description string) *MprisHandler {
	mh := &MprisHandler{
		name:        name,
		description: description,
		msgChan:     make(chan handler.Message),
		ansChan:     make(chan any),
	}

	mh.server = server.NewServer(mh.name, mh, mh)
	mh.evHandler = events.NewEventHandler(mh.server)

	return mh
}

func (mh *MprisHandler) Start(handler func() error) error {
	go mh.server.Listen()

	err := handler()

	mh.evHandler.Player.OnEnded()
	mh.server.Stop()
	mh.mu.Lock()
	mh.closed = true
	mh.mu.Unlock()
	close(mh.msgChan)
	close(mh.ansChan)

	return err
}

func (mh *MprisHandler) send(msg handler.Message) {
	defer func() { recover() }()
	mh.mu.Lock()
	if mh.closed {
		mh.mu.Unlock()
		return
	}
	mh.mu.Unlock()
	mh.msgChan <- msg
}

func (mh *MprisHandler) Message() <-chan handler.Message {
	return mh.msgChan
}

func (mh *MprisHandler) SendAnswer(ans any) {
	mh.ansChan <- ans
}

func (mh *MprisHandler) OnEnded() {
	mh.evHandler.Player.OnEnded()
}

func (mh *MprisHandler) OnVolume() {
	mh.evHandler.Player.OnVolume()
}

func (mh *MprisHandler) OnPlayback(handler.TrackMetadata) {
	mh.evHandler.Player.OnPlayback()
}

func (mh *MprisHandler) OnPlayPause(bool) {
	mh.evHandler.Player.OnPlayPause()
}

func (mh *MprisHandler) OnSeek(position time.Duration) {
	mh.evHandler.Player.OnSeek(types.Microseconds(position.Microseconds()))
}
