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

// A pinned album is stored as a playlist.Item with Kind == ALBUMS but with
// Tracks filled and Albums left empty, and thus SelectedAlbum keeps its zero
// value (0). displayPlaylist must not index into the empty Albums slice.
func TestDisplayPlaylistPinnedAlbumDoesNotPanic(t *testing.T) {
	if err := config.InitialLoad(); err != nil {
		t.Fatalf("config init: %v", err)
	}

	likes := make(map[string]bool)
	cache := make(map[string]bool)

	m := &Model{
		tracklist: tracklist.New(nil, &likes, &cache),
	}

	pl := &playlist.Item{
		Name:    "Pinned Album",
		Kind:    playlist.ALBUMS,
		Active:  true,
		Subitem: true,
		Tracks:  []api.Track{{Id: "1", Title: "track"}},
	}

	m.displayPlaylist(pl)
}

func TestShufflePlaylistCurrentTrackOutOfRange(t *testing.T) {
	config.InitTest(t)
	likes := make(map[string]bool)
	cache := make(map[string]bool)
	playlists := playlist.New(nil)
	pl := &playlist.Item{
		Name:          "user",
		Kind:          playlist.USER,
		Active:        true,
		Tracks:        []api.Track{{Id: "a"}, {Id: "b"}},
		CurrentTrack:  -1,
		SelectedTrack: 0,
	}
	_ = playlists.SetItems([]*playlist.Item{pl})
	playlists.Select(0)

	m := &Model{
		playlists:            playlists,
		tracklist:            tracklist.New(nil, &likes, &cache),
		currentPlaylistIndex: -1,
	}
	_ = m.shufflePlaylist(pl)
	if !m.shuffleOn {
		t.Fatal("shufflePlaylist must set shuffleOn")
	}
	if pl.CurrentTrack != -1 {
		t.Fatalf("CurrentTrack=%d, want -1", pl.CurrentTrack)
	}
	if pl.SelectedTrack < 0 || pl.SelectedTrack >= len(pl.Tracks) || pl.Tracks[pl.SelectedTrack].Id != "a" {
		t.Fatalf("selected track not preserved: index=%d tracks=%v", pl.SelectedTrack, pl.Tracks)
	}
}

func TestAlbumListActiveNilSelection(t *testing.T) {
	config.InitTest(t)
	m := &Model{
		playlists: playlist.New(nil),
	}
	_ = m.playlists.Reset()
	if m.albumListActive() {
		t.Fatal("empty playlist list must not be an album list")
	}
}

func TestRemoveFromPlaylistReturnsCmdImmediately(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	pl := &playlist.Item{
		Name:   "user",
		Kind:   playlist.USER,
		Active: true,
		Tracks: []api.Track{{Id: "a"}, {Id: "b"}},
	}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{pl})
	playlists.Select(0)
	m := &Model{
		playlists:            playlists,
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cache),
		currentPlaylistIndex: -1,
	}

	start := time.Now()
	cmd := m.removeFromPlaylist(pl, 0)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("removeFromPlaylist must not wait for HTTP")
	}
	if cmd == nil {
		t.Fatal("removeFromPlaylist must return Cmd")
	}
	msg := cmd()
	ready, ok := msg.(removePlaylistReadyMsg)
	if !ok || ready.errBanner == "" {
		t.Fatalf("nil client must return error banner, got %#v", msg)
	}
}

func TestApplyRenamePlaylist(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	pl := &playlist.Item{Name: "old", Kind: playlist.USER, Active: true, Revision: 1}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{pl})
	m := &Model{
		playlists:   playlists,
		tracker:     tracker.New(nil, &likes),
		playlistGen: 1,
	}
	m.applyRenamePlaylist(renamePlaylistReadyMsg{gen: 0, title: "skip", revision: 9, index: 0})
	if playlists.Items()[0].Name != "old" {
		t.Fatal("stale renamePlaylistReadyMsg must not rename")
	}
	m.applyRenamePlaylist(renamePlaylistReadyMsg{gen: 1, title: "new", revision: 2, index: 0})
	got := playlists.Items()[0]
	if got.Name != "new" || got.Revision != 2 {
		t.Fatalf("rename: name=%s revision=%d", got.Name, got.Revision)
	}
}

func TestApplyAddPlaylistExisting(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	pl := &playlist.Item{
		Name:     "user",
		Kind:     playlist.USER,
		Active:   true,
		Revision: 1,
		Tracks:   []api.Track{{Id: "old"}},
	}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{pl})
	m := &Model{
		playlists:            playlists,
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cache),
		playlistGen:          1,
		currentPlaylistIndex: -1,
	}
	m.applyAddPlaylist(addPlaylistReadyMsg{
		gen:           1,
		foundIndex:    0,
		addedRevision: 3,
		track:         api.Track{Id: "new"},
	})
	got := playlists.Items()[0]
	if got.Revision != 3 || len(got.Tracks) != 2 || got.Tracks[1].Id != "new" {
		t.Fatalf("add: revision=%d tracks=%v", got.Revision, got.Tracks)
	}
}

func TestApplyRemovePlaylistTrack(t *testing.T) {
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	pl := &playlist.Item{
		Name:     "user",
		Kind:     playlist.USER,
		Active:   true,
		Revision: 1,
		Tracks:   []api.Track{{Id: "a"}, {Id: "b"}},
	}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{pl})
	playlists.Select(0)
	m := &Model{
		playlists:            playlists,
		tracker:              tracker.New(nil, &likes),
		tracklist:            tracklist.New(nil, &likes, &cache),
		playlistGen:          1,
		currentPlaylistIndex: -1,
	}
	m.applyRemovePlaylist(removePlaylistReadyMsg{
		gen:           1,
		listIndex:     0,
		revision:      4,
		tracks:        []api.Track{{Id: "b"}},
		selectedTrack: 0,
		currentTrack:  -1,
	})
	got := playlists.Items()[0]
	if got.Revision != 4 || len(got.Tracks) != 1 || got.Tracks[0].Id != "b" {
		t.Fatalf("remove: revision=%d tracks=%v", got.Revision, got.Tracks)
	}
}
