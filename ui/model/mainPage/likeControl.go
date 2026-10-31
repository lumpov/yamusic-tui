package mainpage

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
)

type likeReadyMsg struct {
	gen          int
	track        api.Track
	unlike       bool
	err          error
	likesIndex   int
	rotorSession string
	rotorBatch   string
}

func (m *Model) likePlayingTrack() tea.Cmd {
	var currentPlaylist *playlist.Item
	if m.currentPlaylistIndex >= 0 {
		currentPlaylist = m.playlists.Items()[m.currentPlaylistIndex]
	}

	track := m.tracker.CurrentTrack()
	return m.likeTrack(track, currentPlaylist)
}

func (m *Model) likeSelectedTrack() tea.Cmd {
	selectedPlaylist := m.playlists.SelectedItem()
	if selectedPlaylist == nil || len(selectedPlaylist.Tracks) == 0 {
		return nil
	}

	track := m.tracklist.SelectedItem().Track
	if track == nil {
		return nil
	}
	return m.likeTrack(track, selectedPlaylist)
}

func (m *Model) likeTrack(track *api.Track, pl *playlist.Item) tea.Cmd {
	if track == nil || m.client == nil {
		return nil
	}

	likedPlaylist, index := m.playlists.GetFirst(playlist.LIKES)
	if likedPlaylist == nil || index < 0 {
		return nil
	}

	m.likeGen++
	gen := m.likeGen
	unlike := m.likedTracksMap[track.Id]
	trackCopy := *track
	var rotorSession, rotorBatch string
	if pl != nil && pl.Rotor {
		rotorSession = pl.SessionId
		rotorBatch = pl.SessionBatch
	}
	client := m.client
	return func() tea.Msg {
		var err error
		if unlike {
			err = client.UnlikeTrack(trackCopy.Id)
		} else {
			err = client.LikeTrack(trackCopy.Id)
		}
		return likeReadyMsg{
			gen:          gen,
			track:        trackCopy,
			unlike:       unlike,
			err:          err,
			likesIndex:   index,
			rotorSession: rotorSession,
			rotorBatch:   rotorBatch,
		}
	}
}

func (m *Model) applyLike(msg likeReadyMsg) tea.Cmd {
	if msg.gen != m.likeGen {
		return nil
	}
	if msg.err != nil {
		m.tracker.ShowError("like")
		return nil
	}

	likedPlaylist, index := m.playlists.GetFirst(playlist.LIKES)
	if likedPlaylist == nil || index < 0 {
		index = msg.likesIndex
		if index < 0 || index >= len(m.playlists.Items()) {
			return nil
		}
		likedPlaylist = m.playlists.Items()[index]
	}

	if msg.unlike {
		delete(m.likedTracksMap, msg.track.Id)
		likedPlaylist.RemoveTrack(msg.track.Id)
	} else {
		m.likedTracksMap[msg.track.Id] = true
		likedPlaylist.AddTrack(&msg.track)
	}

	if msg.rotorSession != "" && m.client != nil {
		evType := api.EV_TRACK_LIKED
		if msg.unlike {
			evType = api.EV_TRACK_UNLIKED
		}
		ev := api.NewTrackFeedbackEvent(evType, &msg.track, 0)
		go m.client.RotorSessionFeedback(msg.rotorSession, api.NewFeedback(msg.rotorBatch, ev))
		log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type+" track: "+msg.track.Title)
	}

	cmd := m.playlists.SetItem(index, likedPlaylist)
	if selected := m.playlists.SelectedItem(); selected != nil && selected.Kind == playlist.LIKES {
		m.displayPlaylist(likedPlaylist)
	}

	m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	return cmd
}
