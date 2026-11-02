package mainpage

import (
	"errors"
	"testing"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
)

func TestDislikePlayingTrackReturnsCmdImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	tr := tracker.New(nil, &likes)
	tr.CurrentTrack().Id = "1"
	tr.CurrentTrack().Title = "t"

	m := &Model{
		client:            &api.YaMusicClient{},
		tracker:           tr,
		playlists:         playlist.New(nil),
		likedTracksMap:    likes,
		dislikedTracksMap: map[string]bool{},
	}

	start := time.Now()
	cmd := m.dislikePlayingTrack()
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("dislikePlayingTrack must not wait for HTTP")
	}
	if cmd == nil {
		t.Fatal("dislikePlayingTrack must return Cmd")
	}

	m.applyDislike(dislikeReadyMsg{gen: m.dislikeGen - 1, track: api.Track{Id: "1"}})
	if m.dislikedTracksMap["1"] {
		t.Fatal("stale dislikeReadyMsg must not change dislikes")
	}
}

func TestDislikePlayingTrackWithoutClientReturnsNil(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	tr := tracker.New(nil, &likes)
	tr.CurrentTrack().Id = "1"
	m := &Model{tracker: tr, dislikedTracksMap: map[string]bool{}}
	if cmd := m.dislikePlayingTrack(); cmd != nil {
		t.Fatal("dislikePlayingTrack without client must be nil")
	}
}

func TestDislikePlayingTrackWithoutTrackReturnsNil(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	m := &Model{
		client:  &api.YaMusicClient{},
		tracker: tracker.New(nil, &likes),
	}
	if cmd := m.dislikePlayingTrack(); cmd != nil {
		t.Fatal("dislikePlayingTrack without current track must be nil")
	}
}

func TestApplyDislikeMarksTrackAndSkips(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{"1": true}
	cache := map[string]bool{}
	likesPl := &playlist.Item{
		Name:   "likes",
		Kind:   playlist.LIKES,
		Active: true,
		Tracks: []api.Track{{Id: "1", Title: "t"}},
	}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{likesPl})

	tr := tracker.New(nil, &likes)
	tr.CurrentTrack().Id = "1"
	tr.CurrentTrack().Available = true

	m := &Model{
		playlists:            playlists,
		tracker:              tr,
		tracklist:            tracklist.New(nil, &likes, &cache),
		likedTracksMap:       likes,
		dislikedTracksMap:    map[string]bool{},
		cachedTracksMap:      cache,
		dislikeGen:           1,
		currentPlaylistIndex: -1,
		playQueue: []api.Track{
			{Id: "1", Title: "t", Available: true},
			{Id: "2", Title: "n", Available: true},
		},
		playQueueTrackIndex: 0,
	}

	cmd := m.applyDislike(dislikeReadyMsg{gen: 1, track: api.Track{Id: "1", Title: "t"}})
	if cmd == nil {
		t.Fatal("applyDislike of current track must skip")
	}
	if !m.dislikedTracksMap["1"] {
		t.Fatal("dislike must set dislikedTracksMap")
	}
	if likes["1"] {
		t.Fatal("dislike must remove like")
	}
	got, _ := playlists.GetFirst(playlist.LIKES)
	if got == nil || len(got.Tracks) != 0 {
		t.Fatalf("likes tracks after dislike: %+v", got)
	}
	if m.playQueueTrackIndex != 1 {
		t.Fatalf("playQueueTrackIndex=%d, want 1", m.playQueueTrackIndex)
	}
}

func TestApplyDislikeErrorDoesNotSkip(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	tr := tracker.New(nil, &likes)
	tr.CurrentTrack().Id = "1"
	m := &Model{
		tracker:              tr,
		dislikedTracksMap:    map[string]bool{},
		dislikeGen:           1,
		playQueue:            []api.Track{{Id: "1", Available: true}, {Id: "2", Available: true}},
		playQueueTrackIndex:  0,
		currentPlaylistIndex: -1,
	}
	cmd := m.applyDislike(dislikeReadyMsg{gen: 1, track: api.Track{Id: "1"}, err: errors.New("fail")})
	if cmd != nil {
		t.Fatal("failed dislike must not skip")
	}
	if m.dislikedTracksMap["1"] {
		t.Fatal("failed dislike must not mark track")
	}
	if m.playQueueTrackIndex != 0 {
		t.Fatalf("playQueueTrackIndex=%d, want 0", m.playQueueTrackIndex)
	}
}
