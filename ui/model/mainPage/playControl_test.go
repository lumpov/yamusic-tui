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
	config.InitTest()
	likes := map[string]bool{}
	m := &Model{
		tracker:              tracker.New(nil, &likes),
		playlists:            playlist.New(nil),
		currentPlaylistIndex: -1,
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
