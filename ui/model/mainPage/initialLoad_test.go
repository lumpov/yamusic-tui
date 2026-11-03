package mainpage

import (
	"testing"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
)

func TestPinnedAlbumItemsDoNotShareTracks(t *testing.T) {
	albums := []api.Album{
		{Title: "first", Volumes: [][]api.Track{{{Id: "a1"}, {Id: "a2"}}}},
		{Title: "second", Volumes: [][]api.Track{{{Id: "b1"}}}},
	}

	items := pinnedAlbumItems(albums)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if got := len(items[0].Tracks); got != 2 {
		t.Errorf("first album: expected 2 tracks, got %d", got)
	}
	if got := len(items[1].Tracks); got != 1 {
		t.Errorf("second album: expected 1 track, got %d (tracks leaked from the previous album)", got)
	}
	if got := items[1].Tracks[0].Id; got != "b1" {
		t.Errorf("second album: expected first track b1, got %s", got)
	}
}

func TestPinnedAlbumItemsSkipEmptyAlbums(t *testing.T) {
	albums := []api.Album{
		{Title: "empty"},
		{Title: "second", Volumes: [][]api.Track{{{Id: "b1"}}}},
	}

	items := pinnedAlbumItems(albums)

	if len(items) != 1 {
		t.Fatalf("expected only the non-empty album, got %d items", len(items))
	}
	if items[0].Name != "second" {
		t.Errorf("expected item %q, got %q", "second", items[0].Name)
	}
}

func TestRotorPlaylistItem(t *testing.T) {
	session := api.StationTracks{
		RadioSessionId: "sid",
		BatchId:        "batch",
		Id:             api.StationId{Type: "user", Tag: "onyourwave"},
		Sequence: []struct {
			Type  string    `json:"type"`
			Track api.Track `json:"track"`
			Liked bool      `json:"liked"`
		}{
			{Type: "track", Track: api.Track{Id: "t1", Title: "song"}},
		},
	}

	item, err := rotorPlaylistItem("Chōchin", playlist.MYWAVE, true, session)
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Chōchin" || item.Kind != playlist.MYWAVE || !item.Rotor || !item.Subitem {
		t.Fatalf("item=%+v", item)
	}
	if item.SessionId != "sid" || item.SessionBatch != "batch" {
		t.Fatalf("session=%s batch=%s", item.SessionId, item.SessionBatch)
	}
	if item.StationId.String() != "user:onyourwave" {
		t.Fatalf("station=%s", item.StationId)
	}
	if len(item.Tracks) != 1 || item.Tracks[0].Id != "t1" {
		t.Fatalf("tracks=%v", item.Tracks)
	}

	if _, err := rotorPlaylistItem("empty", playlist.MYWAVE, true, api.StationTracks{}); err == nil {
		t.Fatal("empty sequence must fail")
	}
}

func TestPinnedWaveMenuItemsHeaderIndented(t *testing.T) {
	waves := []*playlist.Item{
		{Name: "Unfamiliar · Calm", Header: "My Vibe", Kind: playlist.MYWAVE, Active: true, Subitem: true, Rotor: true},
		{Name: "Collection", Header: "My Vibe for", Kind: playlist.MYWAVE, Active: true, Subitem: true, Rotor: true},
		{Name: "same", Header: "same", Kind: playlist.MYWAVE, Active: true, Subitem: true, Rotor: true},
		{Name: "noheader", Kind: playlist.MYWAVE, Active: true, Subitem: true, Rotor: true},
	}

	items := pinnedWaveMenuItems(waves)
	if len(items) != 8 {
		t.Fatalf("len=%d, want 8 (empty, pins:, 3 names + 2 headers + same without dup)", len(items))
	}
	if items[0].Name != "" || items[1].Name != "pins:" {
		t.Fatalf("prefix=%q %q", items[0].Name, items[1].Name)
	}
	if items[2].Name != "Unfamiliar · Calm" || !items[2].Active {
		t.Fatalf("pin=%+v", items[2])
	}
	if items[3].Name != " My Vibe" {
		t.Fatalf("header=%q, want %q", items[3].Name, " My Vibe")
	}
	if items[3].Active || items[3].Kind != playlist.NONE || !items[3].Subitem {
		t.Fatalf("header item=%+v", items[3])
	}
	if items[4].Name != "Collection" || items[5].Name != " My Vibe for" {
		t.Fatalf("second pin=%q header=%q", items[4].Name, items[5].Name)
	}
	if items[6].Name != "same" {
		t.Fatalf("duplicate header still listed as pin: %q", items[6].Name)
	}
	if items[7].Name != "noheader" {
		t.Fatalf("last=%q, want noheader", items[7].Name)
	}
}

func TestPinnedWaveMenuItemsEmpty(t *testing.T) {
	if pinnedWaveMenuItems(nil) != nil {
		t.Fatal("empty waves must not add pins:")
	}
}
