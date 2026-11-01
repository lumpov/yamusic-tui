package mainpage

import (
	"math/rand"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/ui/components/input"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/search"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
	"github.com/dece2183/yamusic-tui/ui/helpers"
	"github.com/dece2183/yamusic-tui/ui/style"
)

type addPlaylistReadyMsg struct {
	gen           int
	errBanner     string
	created       *playlist.Item
	insertIndex   int
	foundIndex    int
	bumpSelect    bool
	bumpCurrent   bool
	sameKind      bool
	addedRevision int
	track         api.Track
}

type renamePlaylistReadyMsg struct {
	gen      int
	err      error
	title    string
	revision int
	index    int
}

type removePlaylistReadyMsg struct {
	gen           int
	errBanner     string
	removeWhole   bool
	listIndex     int
	revision      int
	tracks        []api.Track
	selectedTrack int
	currentTrack  int
	deleteCurrent bool
	playingSame   bool
}

func (m *Model) addPlaylistControl(msg search.Control) tea.Cmd {
	var cmd tea.Cmd

	switch msg {
	case search.SELECT:
		m.isAddPlaylistActive = false

		selectedPlaylist := m.playlists.SelectedItem()
		if selectedPlaylist == nil || len(selectedPlaylist.Tracks) == 0 {
			return nil
		}

		playlists := m.playlists.Items()
		inputVal, ok := m.searchDialog.SuggestionValue()
		if !ok {
			return nil
		}

		foundPlaylistIndex := -1
		var foundPlaylist *playlist.Item
		for i := range playlists {
			if playlists[i].Active && playlists[i].Kind >= playlist.USER {
				if strings.EqualFold(playlists[i].Name, inputVal) {
					foundPlaylist = playlists[i]
					foundPlaylistIndex = i
					break
				} else if foundPlaylistIndex < 0 {
					foundPlaylistIndex = i
				}
			}
		}

		trackIndex := m.tracklist.Index()
		if trackIndex < 0 || trackIndex >= len(selectedPlaylist.Tracks) {
			return nil
		}
		selectedTrack := selectedPlaylist.Tracks[trackIndex]
		selectedKind := selectedPlaylist.Kind
		playlistsIndex := m.playlists.Index()
		currentIndex := m.currentPlaylistIndex
		needCreate := foundPlaylist == nil
		var foundKind uint64
		var foundRevision int
		var foundName string
		var foundLen int
		if foundPlaylist != nil {
			foundKind = foundPlaylist.Kind
			foundRevision = foundPlaylist.Revision
			foundName = foundPlaylist.Name
			foundLen = len(foundPlaylist.Tracks)
		}

		m.playlistGen++
		gen := m.playlistGen
		client := m.client
		return func() tea.Msg {
			if client == nil {
				return addPlaylistReadyMsg{gen: gen, errBanner: "playlist add"}
			}
			msg := addPlaylistReadyMsg{
				gen:         gen,
				insertIndex: foundPlaylistIndex,
				foundIndex:  foundPlaylistIndex,
				track:       selectedTrack,
			}
			if needCreate {
				pl, err := client.CreatePlaylist(inputVal, true)
				if err != nil {
					log.Print(log.LVL_ERROR, "failed to create playlist [%s]: %s", inputVal, err)
					return addPlaylistReadyMsg{gen: gen, errBanner: "playlist create"}
				}
				msg.created = &playlist.Item{
					Name:     pl.Title,
					Kind:     pl.Kind,
					Revision: pl.Revision,
					Active:   true,
					Subitem:  true,
				}
				foundKind = pl.Kind
				foundRevision = pl.Revision
				foundName = pl.Title
				foundLen = 0
				msg.bumpSelect = foundPlaylistIndex < playlistsIndex
				msg.bumpCurrent = currentIndex >= playlistsIndex
			}
			if selectedKind == foundKind {
				msg.sameKind = true
				return msg
			}
			pl, err := client.AddToPlaylist(foundKind, foundRevision, foundLen, selectedTrack.Id)
			if err != nil {
				log.Print(log.LVL_ERROR, "failed to add track [%s] to playlist [%s]: %s", selectedTrack.Id, foundName, err)
				return addPlaylistReadyMsg{gen: gen, errBanner: "playlist add", created: msg.created, insertIndex: msg.insertIndex, bumpSelect: msg.bumpSelect, bumpCurrent: msg.bumpCurrent}
			}
			msg.addedRevision = pl.Revision
			if !needCreate {
				msg.foundIndex = foundPlaylistIndex
			}
			return msg
		}
	case search.CANCEL:
		m.isAddPlaylistActive = false
	case search.UPDATE_SUGGESTIONS:
		inputVal := strings.ToLower(m.searchDialog.InputValue())
		playlists := m.playlists.Items()
		suggestions := make([]string, 0, len(playlists))
		for _, pl := range playlists {
			if !pl.Active || pl.Kind < playlist.USER || (len(inputVal) > 0 && !strings.Contains(strings.ToLower(pl.Name), inputVal)) {
				continue
			}
			suggestions = append(suggestions, pl.Name)
		}
		m.searchDialog.SetSuggestions(suggestions)
	}

	return cmd
}

func (m *Model) applyAddPlaylist(msg addPlaylistReadyMsg) tea.Cmd {
	if msg.gen != m.playlistGen {
		return nil
	}
	if msg.errBanner != "" && msg.created == nil {
		m.tracker.ShowError(msg.errBanner)
		return nil
	}

	foundIndex := msg.foundIndex
	if msg.created != nil {
		m.playlists.InsertItem(msg.insertIndex, msg.created)
		if msg.bumpSelect {
			m.playlists.Select(m.playlists.Index() + 1)
		}
		if msg.bumpCurrent {
			m.currentPlaylistIndex += 1
		}
		foundIndex = msg.insertIndex
	}
	if msg.errBanner != "" {
		m.tracker.ShowError(msg.errBanner)
		return nil
	}
	if msg.sameKind {
		return nil
	}
	items := m.playlists.Items()
	if foundIndex < 0 || foundIndex >= len(items) {
		return nil
	}
	foundPlaylist := items[foundIndex]
	foundPlaylist.Revision = msg.addedRevision
	foundPlaylist.Tracks = append(foundPlaylist.Tracks, msg.track)
	cmd := m.playlists.SetItem(foundIndex, foundPlaylist)
	if m.currentPlaylistIndex >= 0 && foundPlaylist.IsSame(items[m.currentPlaylistIndex]) {
		m.playQueue = foundPlaylist.Tracks
		m.playQueueTrackIndex = foundPlaylist.CurrentTrack
		m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	}
	return cmd
}

func (m *Model) renamePlaylistControl(msg input.Control) tea.Cmd {
	if msg != input.APPLY {
		return nil
	}

	newName := m.inputDialog.Value()
	if len(strings.ReplaceAll(newName, " ", "")) == 0 {
		return nil
	}

	selectedPlaylist := m.playlists.SelectedItem()
	if selectedPlaylist == nil || m.client == nil {
		return nil
	}
	kind := selectedPlaylist.Kind
	oldName := selectedPlaylist.Name
	index := m.playlists.Index()
	m.playlistGen++
	gen := m.playlistGen
	client := m.client
	return func() tea.Msg {
		pl, err := client.RenamePlaylist(kind, newName)
		if err != nil {
			log.Print(log.LVL_ERROR, "failed to rename playlist [%s] to '%s': %s", oldName, newName, err)
			return renamePlaylistReadyMsg{gen: gen, err: err}
		}
		return renamePlaylistReadyMsg{gen: gen, title: pl.Title, revision: pl.Revision, index: index}
	}
}

func (m *Model) applyRenamePlaylist(msg renamePlaylistReadyMsg) tea.Cmd {
	if msg.gen != m.playlistGen {
		return nil
	}
	if msg.err != nil {
		m.tracker.ShowError("playlist rename")
		return nil
	}
	items := m.playlists.Items()
	if msg.index < 0 || msg.index >= len(items) {
		return nil
	}
	selectedPlaylist := items[msg.index]
	selectedPlaylist.Name = msg.title
	selectedPlaylist.Revision = msg.revision
	return m.playlists.SetItem(msg.index, selectedPlaylist)
}

func (m *Model) removeFromPlaylist(pl *playlist.Item, index int) tea.Cmd {
	if pl == nil || index < 0 || index >= len(pl.Tracks) {
		return nil
	}

	switch pl.Kind {
	case playlist.NONE, playlist.MYWAVE, playlist.ALBUMS:
		return nil
	case playlist.LIKES:
		selectedTrack := pl.Tracks[index]
		return m.likeTrack(&selectedTrack, pl)
	case playlist.LOCAL:
		selectedTrack := pl.Tracks[index]
		return m.removeCache(&selectedTrack)
	default:
		kind := pl.Kind
		revision := pl.Revision
		name := pl.Name
		trackId := pl.Tracks[index].Id
		listIndex := m.playlists.Index()
		removeWhole := len(pl.Tracks) < 2
		playingSame := m.currentPlaylistIndex >= 0 && pl.IsSame(m.playlists.Items()[m.currentPlaylistIndex])
		currentTrack := pl.CurrentTrack
		m.playlistGen++
		gen := m.playlistGen
		client := m.client
		return func() tea.Msg {
			if client == nil {
				return removePlaylistReadyMsg{gen: gen, errBanner: "playlist remove"}
			}
			if removeWhole {
				err := client.RemovePlaylist(kind)
				if err != nil {
					log.Print(log.LVL_ERROR, "failed to remove playlist [%s]: %s", name, err)
					return removePlaylistReadyMsg{gen: gen, errBanner: "playlist remove"}
				}
				return removePlaylistReadyMsg{gen: gen, removeWhole: true, listIndex: listIndex}
			}
			newpl, err := client.RemoveFromPlaylist(kind, revision, index)
			if err != nil {
				log.Print(log.LVL_ERROR, "failed to remove track [%s] from playlist [%s]: %s", trackId, name, err)
				return removePlaylistReadyMsg{gen: gen, errBanner: "playlist remove track"}
			}
			tracks := slices.Delete(append([]api.Track(nil), pl.Tracks...), index, index+1)
			selected := index
			if selected >= len(tracks) {
				selected = len(tracks) - 1
			}
			deleteCurrent := index == currentTrack
			cur := currentTrack
			if deleteCurrent {
				cur = -1
			} else if currentTrack > index {
				cur--
			}
			return removePlaylistReadyMsg{
				gen:           gen,
				listIndex:     listIndex,
				revision:      newpl.Revision,
				tracks:        tracks,
				selectedTrack: selected,
				currentTrack:  cur,
				deleteCurrent: deleteCurrent,
				playingSame:   playingSame,
			}
		}
	}
}

func (m *Model) applyRemovePlaylist(msg removePlaylistReadyMsg) tea.Cmd {
	if msg.gen != m.playlistGen {
		return nil
	}
	if msg.errBanner != "" {
		m.tracker.ShowError(msg.errBanner)
		return nil
	}
	if msg.removeWhole {
		if m.currentPlaylistIndex >= msg.listIndex {
			m.currentPlaylistIndex -= 1
		}
		m.playlists.RemoveItem(msg.listIndex)
		if len(m.playlists.Items()) <= m.playlists.Index() {
			m.playlists.Select(0)
		}
		m.displayPlaylist(m.playlists.SelectedItem())
		return nil
	}
	items := m.playlists.Items()
	if msg.listIndex < 0 || msg.listIndex >= len(items) {
		return nil
	}
	pl := items[msg.listIndex]
	pl.Revision = msg.revision
	pl.Tracks = msg.tracks
	pl.SelectedTrack = msg.selectedTrack
	pl.CurrentTrack = msg.currentTrack
	cmd := m.playlists.SetItem(msg.listIndex, pl)
	m.displayPlaylist(pl)
	if msg.playingSame {
		m.playQueue = pl.Tracks
		if msg.deleteCurrent {
			m.playQueueTrackIndex -= 1
		} else {
			m.playQueueTrackIndex = pl.CurrentTrack
		}
	}
	return cmd
}

func (m *Model) shufflePlaylist(pl *playlist.Item) tea.Cmd {
	var cmds []tea.Cmd
	if pl == nil || pl.Kind == playlist.NONE || pl.Kind == playlist.MYWAVE || len(pl.Tracks) == 0 {
		return nil
	}

	currentTrackIndex := pl.CurrentTrack
	selectedTrackIndex := pl.SelectedTrack
	var currentTrack *api.Track
	if currentTrackIndex >= 0 && currentTrackIndex < len(pl.Tracks) {
		currentTrack = &pl.Tracks[currentTrackIndex]
	}
	var selectedTrack *api.Track
	if selectedTrackIndex >= 0 && selectedTrackIndex < len(pl.Tracks) {
		selectedTrack = &pl.Tracks[selectedTrackIndex]
	}

	tracks := make([]api.Track, len(pl.Tracks))
	trackList := make([]tracklist.Item, len(pl.Tracks))
	perm := rand.Perm(len(tracks))

	for i, v := range perm {
		tracks[v] = pl.Tracks[i]
		trackList[v] = tracklist.NewItem(&tracks[v])
		if currentTrack != nil && currentTrack.Id == tracks[v].Id {
			currentTrackIndex = v
		}
		if selectedTrack != nil && selectedTrack.Id == tracks[v].Id {
			selectedTrackIndex = v
		}
	}

	pl.Tracks = tracks
	pl.SelectedTrack = selectedTrackIndex
	pl.CurrentTrack = currentTrackIndex
	cmds = append(cmds, m.playlists.SetItem(m.playlists.Index(), pl))
	cmds = append(cmds, m.tracklist.SetItems(trackList))
	if selectedTrackIndex >= 0 {
		m.tracklist.Select(selectedTrackIndex)
	}

	if m.currentPlaylistIndex >= 0 && pl.IsSame(m.playlists.Items()[m.currentPlaylistIndex]) {
		m.playQueue = pl.Tracks
		m.playQueueTrackIndex = pl.CurrentTrack
		m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	}

	m.shuffleOn = true
	if m.mediaHandler != nil {
		m.mediaHandler.OnShuffle(true)
	}

	return tea.Batch(cmds...)
}

func (m *Model) albumListActive() bool {
	pl := m.playlists.SelectedItem()
	if pl == nil {
		return false
	}
	return len(pl.Albums) > 0 && pl.SelectedAlbum < 0
}

// playSelected starts playback of the track selected in the tracklist (or
// opens the selected album). Used both by the tracklist "play" action and by
// the player play/pause control when nothing is playing yet.
func (m *Model) playSelected() tea.Cmd {
	playlistItem := m.playlists.SelectedItem()
	if playlistItem == nil || !playlistItem.Active {
		return nil
	}
	if m.albumListActive() {
		return m.openAlbum(m.tracklist.Index())
	}
	return m.playSelectedPlaylist(m.tracklist.Index())
}

func (m *Model) openAlbum(index int) tea.Cmd {
	pl := m.playlists.SelectedItem()
	if pl == nil || index < 0 || index >= len(pl.Albums) {
		return nil
	}

	var albumTracks []api.Track
	for _, volume := range pl.Albums[index].Volumes {
		albumTracks = append(albumTracks, volume...)
	}

	var isCurrentAlbum bool
	if m.currentPlaylistIndex > 0 {
		currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
		if currentPlaylist.IsSame(pl) && index == m.currentAlbumIndex {
			isCurrentAlbum = true
		}
	}

	pl.Tracks = albumTracks
	pl.SelectedAlbum = index
	if isCurrentAlbum {
		pl.SelectedTrack = pl.CurrentTrack
	} else {
		pl.SelectedTrack = 0
	}

	m.displayPlaylist(pl)
	m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	return m.playlists.SetItem(m.playlists.Index(), pl)
}

func (m *Model) displayPlaylist(pl *playlist.Item) {
	if pl == nil {
		return
	}
	var itemlist []tracklist.Item
	if len(pl.Albums) > 0 && pl.SelectedAlbum < 0 {
		itemlist = make([]tracklist.Item, len(pl.Albums))
		for i := range pl.Albums {
			itemlist[i] = tracklist.NewAlbumItem(&pl.Albums[i])
		}
	} else {
		itemlist = make([]tracklist.Item, len(pl.Tracks))
		for i := range pl.Tracks {
			itemlist[i] = tracklist.NewItem(&pl.Tracks[i])
		}
		if pl.Rotor && len(itemlist) > 0 {
			itemlist[len(itemlist)-1].IsSuggestion = true
		}
	}

	m.tracklist.SetItems(itemlist)
	m.tracklist.Select(pl.SelectedTrack)

	switch pl.Kind {
	case playlist.MYWAVE:
		m.tracklist.Title = "My wave"
	case playlist.LIKES:
		m.tracklist.Title = "Liked tracks"
	case playlist.LOCAL:
		m.tracklist.Title = "Cached tracks"
	case playlist.ALBUMS:
		if pl.SelectedAlbum >= 0 && pl.SelectedAlbum < len(pl.Albums) {
			album := pl.Albums[pl.SelectedAlbum]
			title := style.TrackTitleStyle.Render(album.Title)
			artists := style.TrackArtistStyle.Render(" (" + helpers.ArtistList(album.Artists) + ")")
			m.tracklist.Title = "Tracks from " + title + artists
		} else {
			if pl.Name == "albums" {
				m.tracklist.Title = "Liked albums"
			} else {
				m.tracklist.Title = pl.Name
			}
		}
	default:
		title := style.TrackTitleStyle.Render(pl.Name)
		m.tracklist.Title = "Tracks from " + title
	}
}

func (m *Model) indicateCurrentTrackPlaying(playing bool) {
	if m.currentPlaylistIndex < 0 {
		return
	}

	currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
	if !currentPlaylist.IsSame(m.playlists.SelectedItem()) || currentPlaylist.CurrentTrack < 0 {
		return
	}

	if len(currentPlaylist.Albums) > 0 && m.currentAlbumIndex != currentPlaylist.SelectedAlbum {
		if currentPlaylist.SelectedAlbum < 0 {
			albums := m.tracklist.Items()
			if m.currentAlbumIndex > -1 && m.currentAlbumIndex < len(albums) {
				album := albums[m.currentAlbumIndex]
				album.IsPlaying = playing
				m.tracklist.SetItem(m.currentAlbumIndex, album)
			}
		}
		return
	}

	tracks := m.tracklist.Items()
	if currentPlaylist.CurrentTrack < len(tracks) {
		track := tracks[currentPlaylist.CurrentTrack]
		track.IsPlaying = playing
		m.tracklist.SetItem(currentPlaylist.CurrentTrack, track)
	}
}
