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

type stubMedia struct{}

func (stubMedia) Start(func() error) error         { return nil }
func (stubMedia) Message() <-chan handler.Message  { return nil }
func (stubMedia) SendAnswer(any)                   {}
func (stubMedia) OnEnded()                         {}
func (stubMedia) OnVolume()                        {}
func (stubMedia) OnPlayback(handler.TrackMetadata) {}
func (stubMedia) OnPlayPause(bool)                 {}
func (stubMedia) OnSeek(time.Duration)             {}

func pausedPlaylistModel(t *testing.T) *Model {
	t.Helper()
	config.InitTest(t)
	likes := map[string]bool{}
	cache := map[string]bool{}
	track := api.Track{Id: "1", Title: "t"}
	pl := &playlist.Item{
		Name:          "likes",
		Kind:          playlist.LIKES,
		Active:        true,
		Tracks:        []api.Track{track},
		CurrentTrack:  0,
		SelectedTrack: 0,
		SelectedAlbum: -1,
	}
	playlists := playlist.New(nil)
	_ = playlists.SetItems([]*playlist.Item{pl})
	playlists.Select(0)
	tl := tracklist.New(nil, &likes, &cache)
	_ = tl.SetItems([]tracklist.Item{{Track: &track, IsPlaying: true}})
	return &Model{
		playlists:            playlists,
		tracklist:            tl,
		tracker:              tracker.New(nil, &likes),
		currentPlaylistIndex: 0,
		currentAlbumIndex:    -1,
		uiFocused:            true,
		mediaHandler:         stubMedia{},
	}
}

func TestPlaylistCursorDoesNotForcePlayingIcon(t *testing.T) {
	m := pausedPlaylistModel(t)
	next, _ := m.Update(playlist.CURSOR_UP)
	got := next.(*Model)
	if got.tracklist.Items()[0].IsPlaying {
		t.Fatal("paused tracker must not show play icon after playlist cursor")
	}
}

func TestTracklistBackDoesNotForcePlayingIcon(t *testing.T) {
	m := pausedPlaylistModel(t)
	pl := m.playlists.SelectedItem()
	pl.Albums = []api.Album{{Title: "a"}}
	pl.SelectedAlbum = 0
	_ = m.playlists.SetItem(0, pl)
	next, _ := m.Update(tracklist.BACK)
	got := next.(*Model)
	if len(got.tracklist.Items()) > 0 && got.tracklist.Items()[0].IsPlaying {
		t.Fatal("paused tracker must not show play icon after Back")
	}
}

func TestPauseClearsPlayingIcon(t *testing.T) {
	m := pausedPlaylistModel(t)
	next, _ := m.Update(tracker.PAUSE)
	got := next.(*Model)
	if got.tracklist.Items()[0].IsPlaying {
		t.Fatal("Pause must clear play icon")
	}
}
