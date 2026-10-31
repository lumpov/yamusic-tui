package mainpage

import (
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
)

func TestLikeTrackReturnsCmdImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	likesPl := &playlist.Item{Name: "likes", Kind: playlist.LIKES, Active: true}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{likesPl})

	m := &Model{
		client:          &api.YaMusicClient{},
		playlists:       playlists,
		tracker:         tracker.New(nil, &likes),
		tracklist:       tracklist.New(nil, &likes, &cache),
		likedTracksMap:  likes,
		cachedTracksMap: cache,
	}

	start := time.Now()
	cmd := m.likeTrack(&api.Track{Id: "1", Title: "t"}, nil)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("likeTrack must not wait for HTTP")
	}
	if cmd == nil {
		t.Fatal("likeTrack must return Cmd")
	}

	m.applyLike(likeReadyMsg{gen: m.likeGen - 1, track: api.Track{Id: "1"}})
	if likes["1"] {
		t.Fatal("stale likeReadyMsg must not change likes")
	}
}

func TestLikeTrackWithoutClientReturnsNil(t *testing.T) {
	config.InitTest(t)
	likesPl := &playlist.Item{Name: "likes", Kind: playlist.LIKES, Active: true}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{likesPl})
	m := &Model{playlists: playlists, likedTracksMap: map[string]bool{}}
	if cmd := m.likeTrack(&api.Track{Id: "1"}, nil); cmd != nil {
		t.Fatal("likeTrack without client must be nil")
	}
}

func TestApplyLikeAddsTrack(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	likesPl := &playlist.Item{Name: "likes", Kind: playlist.LIKES, Active: true}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{likesPl})
	playlists.Select(0)

	m := &Model{
		playlists:            playlists,
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cache),
		likedTracksMap:       likes,
		cachedTracksMap:      cache,
		likeGen:              1,
		currentPlaylistIndex: -1,
	}
	m.applyLike(likeReadyMsg{gen: 1, track: api.Track{Id: "1", Title: "t"}})
	if !likes["1"] {
		t.Fatal("like must set likedTracksMap")
	}
	got, _ := playlists.GetFirst(playlist.LIKES)
	if got == nil || len(got.Tracks) != 1 || got.Tracks[0].Id != "1" {
		t.Fatalf("likes tracks: %+v", got)
	}
}
