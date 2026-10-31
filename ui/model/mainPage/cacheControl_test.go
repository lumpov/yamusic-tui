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

func TestCacheCurrentTrackStoppedReturnsNil(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cacheMap := map[string]bool{}
	m := &Model{
		tracker:         tracker.New(nil, &likes),
		cachedTracksMap: cacheMap,
	}
	start := time.Now()
	if cmd := m.cacheCurrentTrack(); cmd != nil {
		t.Fatal("stopped tracker must not cache")
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("cacheCurrentTrack must not wait for disk")
	}
}

func TestRemoveCacheReturnsCmdImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cacheMap := map[string]bool{"1": true}
	local := &playlist.Item{Name: "local", Kind: playlist.LOCAL, Active: true, Tracks: []api.Track{{Id: "1"}}}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{local})
	m := &Model{
		tracker:         tracker.New(nil, &likes),
		playlists:       playlists,
		cachedTracksMap: cacheMap,
	}

	start := time.Now()
	cmd := m.removeCache(&api.Track{Id: "1"})
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("removeCache must not wait for disk")
	}
	if cmd == nil {
		t.Fatal("removeCache must return Cmd")
	}

	m.applyCacheRemove(cacheRemoveReadyMsg{gen: m.cacheGen - 1, trackId: "1"})
	if !cacheMap["1"] {
		t.Fatal("stale cacheRemoveReadyMsg must not remove")
	}
}

func TestApplyCacheAddsTrack(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cacheMap := map[string]bool{}
	local := &playlist.Item{Name: "local", Kind: playlist.LOCAL, Active: true}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{local})
	playlists.Select(0)
	m := &Model{
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cacheMap),
		playlists:            playlists,
		cachedTracksMap:      cacheMap,
		cacheGen:             1,
		currentPlaylistIndex: -1,
	}
	m.applyCache(cacheReadyMsg{gen: 1, track: api.Track{Id: "1", Title: "t"}})
	if !cacheMap["1"] {
		t.Fatal("cache must set cachedTracksMap")
	}
	got, _ := playlists.GetFirst(playlist.LOCAL)
	if got == nil || len(got.Tracks) != 1 || got.Tracks[0].Id != "1" {
		t.Fatalf("local tracks: %+v", got)
	}
}

func TestApplyCacheRemoveDeletesTrack(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cacheMap := map[string]bool{"1": true}
	local := &playlist.Item{Name: "local", Kind: playlist.LOCAL, Active: true, Tracks: []api.Track{{Id: "1"}, {Id: "2"}}}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{local})
	playlists.Select(0)
	m := &Model{
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cacheMap),
		playlists:            playlists,
		cachedTracksMap:      cacheMap,
		cacheGen:             1,
		currentPlaylistIndex: -1,
	}
	m.applyCacheRemove(cacheRemoveReadyMsg{gen: 1, trackId: "1"})
	if cacheMap["1"] {
		t.Fatal("remove must clear cachedTracksMap")
	}
	got, _ := playlists.GetFirst(playlist.LOCAL)
	if got == nil || len(got.Tracks) != 1 || got.Tracks[0].Id != "2" {
		t.Fatalf("local tracks: %+v", got)
	}
}
