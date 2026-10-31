package mainpage

import (
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/media/handler"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
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
	if got, ok := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SET_SHUFFLE, Arg: true}).(mediaShuffleMsg); !ok || !got.on {
		t.Fatal("SET_SHUFFLE true must be mediaShuffleMsg on")
	}
	if got, ok := mediaPlaybackMsg(handler.Message{Type: handler.MSG_SET_SHUFFLE, Arg: false}).(mediaShuffleMsg); !ok || got.on {
		t.Fatalf("SET_SHUFFLE false: got %v", got)
	}
	if got, ok := mediaPlaybackMsg(handler.Message{Type: handler.MSG_OPEN_URI, Arg: "456"}).(mediaOpenUriMsg); !ok || got.trackId != "456" {
		t.Fatalf("OPEN_URI: got %v", got)
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
	shuffleN    int
	lastShuffle bool
	volumeN     int
	lastVolume  float64
}

func (c *captureMedia) OnShuffle(on bool) {
	c.shuffleN++
	c.lastShuffle = on
}

func (c *captureMedia) OnVolume(vol float64) {
	c.volumeN++
	c.lastVolume = vol
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
	_, _ = got.Update(handler.Message{Type: handler.MSG_GET_SHUFFLE})
	if cap.ans != false {
		t.Fatalf("GET_SHUFFLE default: %v", cap.ans)
	}
	got.shuffleOn = true
	_, _ = got.Update(handler.Message{Type: handler.MSG_GET_SHUFFLE})
	if cap.ans != true {
		t.Fatalf("GET_SHUFFLE on: %v", cap.ans)
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

func TestShuffleOffClearsFlag(t *testing.T) {
	config.InitTest(t)
	cap := &captureMedia{}
	m := &Model{
		mediaHandler: cap,
		uiFocused:    true,
		shuffleOn:    true,
	}
	next, _ := m.Update(mediaShuffleMsg{on: false})
	got := next.(*Model)
	if got.shuffleOn {
		t.Fatal("SET_SHUFFLE false must clear shuffleOn")
	}
	if cap.shuffleN != 1 || cap.lastShuffle {
		t.Fatalf("OnShuffle: n=%d on=%v", cap.shuffleN, cap.lastShuffle)
	}
}

func TestVolumeToCallsOnVolume(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cap := &captureMedia{}
	m := &Model{
		tracker:      tracker.New(nil, &likes),
		mediaHandler: cap,
		uiFocused:    true,
	}
	_, _ = m.Update(tracker.VolumeTo(0.4))
	if cap.volumeN != 1 || cap.lastVolume != 0.4 {
		t.Fatalf("OnVolume n=%d vol=%v", cap.volumeN, cap.lastVolume)
	}
}

func TestPlayLoadedTrackIdSelectsTrack(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	playlists := playlist.New(nil)
	pl := &playlist.Item{
		Name:          "likes",
		Kind:          playlist.LIKES,
		Active:        true,
		Tracks:        []api.Track{{Id: "a"}, {Id: "b", Title: "second"}},
		SelectedTrack: 0,
		SelectedAlbum: -1,
	}
	_ = playlists.SetItems([]*playlist.Item{pl})
	m := &Model{
		playlists:            playlists,
		tracklist:            tracklist.New(nil, &likes, &cache),
		tracker:              tracker.New(nil, &likes),
		uiFocused:            true,
		currentPlaylistIndex: -1,
	}
	cmd := m.playLoadedTrackId("b")
	if cmd == nil {
		t.Fatal("playLoadedTrackId must start play")
	}
	if m.playlists.Index() != 0 || m.playlists.SelectedItem().SelectedTrack != 1 {
		t.Fatalf("selected playlist/track: index=%d track=%d", m.playlists.Index(), m.playlists.SelectedItem().SelectedTrack)
	}
	if len(m.playQueue) != 2 || m.playQueueTrackIndex != 1 || m.playQueue[1].Id != "b" {
		t.Fatalf("playQueue=%v idx=%d", m.playQueue, m.playQueueTrackIndex)
	}
	if m.playLoadedTrackId("missing") != nil {
		t.Fatal("unknown track must not play")
	}
}
