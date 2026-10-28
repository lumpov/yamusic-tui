package mainpage

import (
	"testing"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
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
	config.InitTest()
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
	if pl.CurrentTrack != -1 {
		t.Fatalf("CurrentTrack=%d, want -1", pl.CurrentTrack)
	}
	if pl.SelectedTrack < 0 || pl.SelectedTrack >= len(pl.Tracks) || pl.Tracks[pl.SelectedTrack].Id != "a" {
		t.Fatalf("selected track not preserved: index=%d tracks=%v", pl.SelectedTrack, pl.Tracks)
	}
}

func TestAlbumListActiveNilSelection(t *testing.T) {
	config.InitTest()
	m := &Model{
		playlists: playlist.New(nil),
	}
	_ = m.playlists.Reset()
	if m.albumListActive() {
		t.Fatal("empty playlist list must not be an album list")
	}
}
