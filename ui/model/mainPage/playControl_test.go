package mainpage

import (
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
)

func TestPlayTrackReturnsCmdImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	m := &Model{
		tracker:              tracker.New(nil, &likes),
		playlists:            playlist.New(nil),
		currentPlaylistIndex: -1,
		mediaHandler:         stubMedia{},
	}

	start := time.Now()
	cmd := m.playTrack(&api.Track{Id: "1", Title: "t"})
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("playTrack must not wait for HTTP")
	}
	if cmd == nil {
		t.Fatal("playTrack must return Cmd")
	}

	msg := playTrackReadyMsg{gen: m.playGen - 1, err: nil}
	m.applyPlayTrack(msg)
	if m.tracker.CurrentTrack() != nil && m.tracker.CurrentTrack().Id == "1" {
		t.Fatal("stale playTrackReadyMsg must not start the track")
	}
}

func TestRotateTracksCmdReturnsImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	pl := &playlist.Item{
		Name:      "rotor",
		Rotor:     true,
		SessionId: "s1",
		Tracks:    []api.Track{{Id: "1"}},
	}
	m := &Model{
		tracker: tracker.New(nil, &likes),
	}

	start := time.Now()
	if cmd := m.rotateTracksCmd(pl); cmd != nil {
		t.Fatal("rotateTracksCmd without client must be nil")
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("rotateTracksCmd must not wait for HTTP")
	}
}
