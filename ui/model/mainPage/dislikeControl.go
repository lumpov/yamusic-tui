package mainpage

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
)

type dislikeReadyMsg struct {
	gen           int
	track         api.Track
	err           error
	likesIndex    int
	playedSeconds float64
	rotorSession  string
	rotorBatch    string
}

func (m *Model) dislikePlayingTrack() tea.Cmd {
	if m.tracker == nil || m.client == nil {
		return nil
	}

	track := m.tracker.CurrentTrack()
	if track == nil || track.Id == "" {
		return nil
	}

	var currentPlaylist *playlist.Item
	if m.playlists != nil && m.currentPlaylistIndex >= 0 {
		items := m.playlists.Items()
		if m.currentPlaylistIndex < len(items) {
			currentPlaylist = items[m.currentPlaylistIndex]
		}
	}

	likesIndex := -1
	if m.playlists != nil {
		_, likesIndex = m.playlists.GetFirst(playlist.LIKES)
	}

	m.dislikeGen++
	gen := m.dislikeGen
	trackCopy := *track
	played := m.tracker.Playtime().Seconds()
	var rotorSession, rotorBatch string
	if currentPlaylist != nil && currentPlaylist.Rotor {
		rotorSession = currentPlaylist.SessionId
		rotorBatch = currentPlaylist.SessionBatch
	}
	client := m.client
	return func() tea.Msg {
		err := client.DislikeTrack(trackCopy.Id)
		return dislikeReadyMsg{
			gen:           gen,
			track:         trackCopy,
			err:           err,
			likesIndex:    likesIndex,
			playedSeconds: played,
			rotorSession:  rotorSession,
			rotorBatch:    rotorBatch,
		}
	}
}

func (m *Model) applyDislike(msg dislikeReadyMsg) tea.Cmd {
	if msg.gen != m.dislikeGen {
		return nil
	}
	if msg.err != nil {
		log.Print(log.LVL_ERROR, "failed to dislike track [%s]: %s", msg.track.Id, msg.err)
		m.tracker.ShowError("dislike")
		return nil
	}

	if m.dislikedTracksMap == nil {
		m.dislikedTracksMap = make(map[string]bool)
	}
	m.dislikedTracksMap[msg.track.Id] = true
	delete(m.likedTracksMap, msg.track.Id)

	if m.playlists != nil {
		likedPlaylist, index := m.playlists.GetFirst(playlist.LIKES)
		if likedPlaylist == nil || index < 0 {
			index = msg.likesIndex
			if index >= 0 && index < len(m.playlists.Items()) {
				likedPlaylist = m.playlists.Items()[index]
			}
		}
		if likedPlaylist != nil && index >= 0 {
			likedPlaylist.RemoveTrack(msg.track.Id)
			_ = m.playlists.SetItem(index, likedPlaylist)
			if selected := m.playlists.SelectedItem(); selected != nil && selected.Kind == playlist.LIKES {
				m.displayPlaylist(likedPlaylist)
			}
		}
	}

	if msg.rotorSession != "" && m.client != nil {
		ev := api.NewTrackFeedbackEvent(api.EV_TRACK_DISLIKED, &msg.track, msg.playedSeconds)
		go m.client.RotorSessionFeedback(msg.rotorSession, api.NewFeedback(msg.rotorBatch, ev))
		log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type+" track: "+msg.track.Title)
	}

	if m.tracker != nil {
		m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
		if curr := m.tracker.CurrentTrack(); curr != nil && curr.Id == msg.track.Id {
			return m.nextTrack()
		}
	}
	return nil
}
