package mainpage

import (
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/media/handler"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
)

func TestMediaPlaybackMsgPlayPauseSeekVolume(t *testing.T) {
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_PLAY}); got != tracker.PLAY {
		t.Fatalf("PLAY: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_PAUSE}); got != tracker.PAUSE {
		t.Fatalf("PAUSE: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_PLAYPAUSE}); got != tracker.PLAYPAUSE {
		t.Fatalf("PLAYPAUSE: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_NEXT}); got != tracker.NEXT {
		t.Fatalf("NEXT: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_PREVIOUS}); got != tracker.PREV {
		t.Fatalf("PREV: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_STOP}); got != tracker.STOP {
		t.Fatalf("STOP: got %v", got)
	}

	seek := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SEEK, Arg: 5 * time.Second})
	if seek != tracker.SeekBy(5*time.Second) {
		t.Fatalf("SEEK: got %v", seek)
	}
	pos := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SETPOS, Arg: 2 * time.Second})
	if pos != tracker.SeekTo(2*time.Second) {
		t.Fatalf("SETPOS: got %v", pos)
	}
	vol := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SET_VOLUME, Arg: 0.4})
	if vol != tracker.VolumeTo(0.4) {
		t.Fatalf("SET_VOLUME: got %v", vol)
	}
	if _, ok := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SET_SHUFFLE, Arg: true}).(mediaShuffleMsg); !ok {
		t.Fatal("SET_SHUFFLE true must be mediaShuffleMsg")
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SET_SHUFFLE, Arg: false}); got != nil {
		t.Fatalf("SET_SHUFFLE false: got %v", got)
	}
	if got := mediaPlaybackMsg(handler.Message{Type: handler.MSG_GET_VOLUME}); got != nil {
		t.Fatalf("GET must not map to a playback message, got %v", got)
	}
}

type captureMedia struct {
	stubMedia
	ans         any
	playPauseN  int
	lastPlaying bool
}

func (c *captureMedia) SendAnswer(v any) { c.ans = v }

func (c *captureMedia) OnPlayPause(playing bool) {
	c.playPauseN++
	c.lastPlaying = playing
}

func TestAnswerMediaGetOnUpdate(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cap := &captureMedia{}
	m := &Model{
		tracker:      tracker.New(nil, &likes),
		mediaHandler: cap,
		uiFocused:    true,
	}

	next, _ := m.Update(handler.Message{Type: handler.MSG_GET_PLAYBACKSTATUS})
	got := next.(*Model)
	if cap.ans != handler.STATE_STOPED {
		t.Fatalf("GET_PLAYBACKSTATUS: got %v", cap.ans)
	}
	next, _ = got.Update(handler.Message{Type: handler.MSG_GET_VOLUME})
	got = next.(*Model)
	if _, ok := cap.ans.(float64); !ok {
		t.Fatalf("GET_VOLUME type %T", cap.ans)
	}
	_, _ = got.Update(handler.Message{Type: handler.MSG_GET_METADATA})
	md, ok := cap.ans.(handler.TrackMetadata)
	if !ok || md.TrackId != "" {
		t.Fatalf("stopped GET_METADATA: %+v", cap.ans)
	}
}

func TestPauseDoesNotNotifyMediaWhenStopped(t *testing.T) {
	m := pausedPlaylistModel(t)
	cap := &captureMedia{}
	m.mediaHandler = cap
	_, _ = m.Update(tracker.PAUSE)
	if cap.playPauseN != 0 {
		t.Fatalf("stopped tracker must not call OnPlayPause, n=%d", cap.playPauseN)
	}
}
