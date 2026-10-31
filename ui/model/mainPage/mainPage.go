package mainpage

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/media/handler"
	"github.com/dece2183/yamusic-tui/ui/components/input"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/search"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
	"github.com/dece2183/yamusic-tui/ui/model"
	"github.com/dece2183/yamusic-tui/ui/style"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/go-clipboard"
)

type Model struct {
	program       *tea.Program
	client        *api.YaMusicClient
	clipboard     *clipboard.Clipboard
	mediaHandler  handler.MediaHandler
	width, height int

	spinner   spinner.Model
	playlists *playlist.Model
	tracklist *tracklist.Model
	tracker   *tracker.Model

	searchDialog           *search.Model
	inputDialog            *input.Model
	isLoading              bool
	isSearchActive         bool
	isAddPlaylistActive    bool
	isRenamePlaylistActive bool
	isPlaylistHideOverride bool
	uiFocused              bool
	idleView               string

	playQueue            []api.Track
	playQueueTrackIndex  int
	playGen              int
	currentPlaylistIndex int
	currentAlbumIndex    int

	likedTracksMap  map[string]bool
	cachedTracksMap map[string]bool
}

// mainpage.Model constructor.
func New(mediaHandler handler.MediaHandler) *Model {
	m := &Model{}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithReportFocus())
	m.program = p
	m.clipboard = clipboard.New()
	m.mediaHandler = mediaHandler
	m.likedTracksMap = make(map[string]bool)
	m.cachedTracksMap = make(map[string]bool)
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Points))
	m.playlists = playlist.New(m.program)
	m.tracklist = tracklist.New(m.program, &m.likedTracksMap, &m.cachedTracksMap)
	m.tracker = tracker.New(m.program, &m.likedTracksMap)
	m.searchDialog = search.New()
	m.inputDialog = input.New()
	m.uiFocused = true

	return m
}

//
// model.Model interface implementation
//

func (m *Model) Run() error {
	go m.mediaHandle()
	m.removeCoverFiles()
	_, err := m.program.Run()
	m.tracker.Stop()
	m.removeCoverFiles()
	return err
}

func (m *Model) Send(msg tea.Msg) {
	go m.program.Send(msg)
}

//
// tea.Model interface implementation
//

func (m *Model) Init() tea.Cmd {
	m.isLoading = true
	go m.initialLoad()
	return tea.Batch(tea.SetWindowTitle(config.DirName), m.spinner.Tick)
}

type mediaShuffleMsg struct{}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if skipUIProgress(m.uiFocused, message) {
		return m, nil
	}

	switch msg := message.(type) {
	case tea.FocusMsg:
		m.uiFocused = true
		m.idleView = ""
		if m.tracker != nil && !m.tracker.IsStoped() {
			m.tracker, cmd = m.tracker.Update(tracker.ProgressControl(m.tracker.Progress()))
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case tea.BlurMsg:
		m.uiFocused = false
		m.idleView = ""
		return m, nil
	case LoadingMsg:
		m.isLoading = false
		return m, model.Cmd(playlist.CURSOR_UP)
	case playTrackReadyMsg:
		m.applyPlayTrack(msg)
		return m, nil
	case rotateTracksReadyMsg:
		m.applyRotateTracks(msg)
		return m, nil
	case searchErrMsg:
		m.isLoading = false
		m.tracker.ShowError(msg.text)
		return m, nil
	case searchSuggestionsMsg:
		m.searchDialog.SetSuggestions(msg.list)
		return m, nil
	case searchResultsReadyMsg:
		return m, m.applySearchResults(msg)
	case handler.Message:
		m.answerMediaGet(msg)
		return m, nil

	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		m.idleView = ""
		return m, tea.ClearScreen

	case tea.KeyMsg:
		controls := config.Current.Controls
		keypress := msg.String()

		switch {
		case controls.Quit.Contains(keypress):
			return m, tea.Quit
		case m.isSearchActive || m.isAddPlaylistActive:
			m.searchDialog, cmd = m.searchDialog.Update(message)
			cmds = append(cmds, cmd)
		case m.isRenamePlaylistActive:
			m.inputDialog, cmd = m.inputDialog.Update(message)
			cmds = append(cmds, cmd)
		case controls.Reload.Contains(keypress):
			m.isLoading = true
			cmd = m.playlists.Reset()
			cmds = append(cmds, cmd)
			cmds = append(cmds, m.spinner.Tick)
			go m.initialLoad()
		default:
			if m.isLoading {
				m.spinner, cmd = m.spinner.Update(message)
				cmds = append(cmds, cmd)
			} else {
				m.playlists, cmd = m.playlists.Update(message)
				cmds = append(cmds, cmd)
				m.tracklist, cmd = m.tracklist.Update(message)
				cmds = append(cmds, cmd)
				m.tracker, cmd = m.tracker.Update(message)
				cmds = append(cmds, cmd)
			}
		}

	// playlist control update
	case playlist.Control:
		switch msg {
		case playlist.CURSOR_UP, playlist.CURSOR_DOWN:
			selectedPlaylist := m.playlists.SelectedItem()
			if selectedPlaylist == nil {
				break
			}

			items := m.playlists.Items()
			if m.currentPlaylistIndex >= 0 && m.currentPlaylistIndex < len(items) {
				currentPlaylist := items[m.currentPlaylistIndex]
				if selectedPlaylist.IsSame(currentPlaylist) && len(selectedPlaylist.Tracks) > 0 && m.currentAlbumIndex == selectedPlaylist.SelectedAlbum {
					selectedPlaylist.SelectedTrack = selectedPlaylist.CurrentTrack
					m.playlists.SetItem(m.playlists.Index(), selectedPlaylist)
				}
			} else {
				m.currentPlaylistIndex = -1
			}

			m.displayPlaylist(selectedPlaylist)
			m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())

			m.tracklist.Shufflable = (selectedPlaylist.Kind != playlist.NONE && selectedPlaylist.Kind != playlist.MYWAVE && len(selectedPlaylist.Tracks) > 0)
		case playlist.RENAME:
			selectedPlaylist := m.playlists.SelectedItem()
			if selectedPlaylist == nil || selectedPlaylist.Kind < playlist.USER {
				break
			}
			m.inputDialog.Title = "Rename playlist " + selectedPlaylist.Name
			m.inputDialog.SetValue(selectedPlaylist.Name)
			m.isRenamePlaylistActive = true
		case playlist.TOGGLE_VIEW:
			m.isPlaylistHideOverride = !m.isPlaylistHideOverride
		}

	// tracklist control update
	case tracklist.Control:
		switch msg {
		case tracklist.PLAY:
			cmd = m.playSelected()
			cmds = append(cmds, cmd)
		case tracklist.CURSOR_UP, tracklist.CURSOR_DOWN:
			currentPlaylist := m.playlists.SelectedItem()
			if currentPlaylist == nil {
				break
			}
			cursorIndex := m.tracklist.Index()
			currentPlaylist.SelectedTrack = cursorIndex
			cmd = m.playlists.SetItem(m.playlists.Index(), currentPlaylist)
			cmds = append(cmds, cmd)
		case tracklist.LIKE:
			if !m.albumListActive() {
				cmd = m.likeSelectedTrack()
				cmds = append(cmds, cmd)
			}
		case tracklist.SHOW_ARTIST:
			if m.albumListActive() {
				break
			}
			track := m.tracklist.SelectedItem()
			if track.Track == nil {
				break
			}
			searchRes := api.SearchResult{
				Artists: api.SearchResultArtists{
					Results: track.Track.Artists,
				},
			}
			searchConf := config.Search{
				Artists: true,
			}
			m.isLoading = true
			cmds = append(cmds, m.spinner.Tick, m.displaySearchResults(searchRes, &searchConf))
		case tracklist.ADD_TO_PLAYLIST:
			if m.albumListActive() {
				break
			}
			selectedTrack := m.tracklist.SelectedItem()
			if selectedTrack.Track == nil {
				break
			}
			m.searchDialog.Title = "Add " + selectedTrack.Track.Title + " to"
			m.searchDialog.Action = "add"
			m.isAddPlaylistActive = true
			m.Send(search.UPDATE_SUGGESTIONS)
		case tracklist.REMOVE_FROM_PLAYLIST:
			selectedPlaylist := m.playlists.SelectedItem()
			if selectedPlaylist == nil {
				break
			}
			cmd = m.removeFromPlaylist(selectedPlaylist, m.tracklist.Index())
			cmds = append(cmds, cmd)
		case tracklist.SEARCH:
			m.searchDialog.Title = "Search"
			m.searchDialog.Action = "search"
			m.isSearchActive = true
			m.Send(search.UPDATE_SUGGESTIONS)
		case tracklist.SHUFFLE:
			cmd = m.shufflePlaylist(m.playlists.SelectedItem())
			cmds = append(cmds, cmd)
		case tracklist.SHARE:
			if m.albumListActive() {
				break
			}
			link := api.ShareTrackLink(m.tracklist.SelectedItem().Track)
			if link != "" {
				m.clipboard.CopyText(link)
			}
		case tracklist.BACK:
			selectedPlaylist := m.playlists.SelectedItem()
			if selectedPlaylist == nil {
				break
			}
			if len(selectedPlaylist.Albums) > 0 && selectedPlaylist.SelectedAlbum >= 0 {
				selectedPlaylist.SelectedTrack = selectedPlaylist.SelectedAlbum
				selectedPlaylist.SelectedAlbum = -1
				m.displayPlaylist(selectedPlaylist)
				m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
				cmd = m.playlists.SetItem(m.playlists.Index(), selectedPlaylist)
				cmds = append(cmds, cmd)
			}
		}

	// player control update
	case tracker.Control:
		switch msg {
		case tracker.NEXT:
			cmds = append(cmds, m.nextTrack())
		case tracker.PREV:
			cmds = append(cmds, m.prevTrack())
		case tracker.LIKE:
			cmd = m.likePlayingTrack()
			cmds = append(cmds, cmd)
		case tracker.PLAY:
			if m.tracker.IsStoped() {
				cmd = m.playSelected()
				cmds = append(cmds, cmd)
			}
		case tracker.PLAYPAUSE:
			if m.tracker.IsStoped() {
				cmd = m.playSelected()
				cmds = append(cmds, cmd)
			}
		case tracker.STOP:
			m.mediaHandler.OnEnded()
		case tracker.REWIND:
			m.mediaHandler.OnSeek(m.tracker.Position())
		case tracker.VOLUME:
			m.mediaHandler.OnVolume()
		case tracker.CACHE_TRACK:
			cmd = m.cacheCurrentTrack()
			cmds = append(cmds, cmd)
		case tracker.BUFFERING_COMPLETE:
			cacheMode := config.Current.CacheTracks
			if cacheMode == config.CACHE_ALL || (cacheMode == config.CACHE_LIKED_ONLY && m.likedTracksMap[m.tracker.CurrentTrack().Id]) {
				cmd = m.cacheCurrentTrack()
				cmds = append(cmds, cmd)
			}
		}

		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)
		m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
		switch msg {
		case tracker.PLAY, tracker.PAUSE, tracker.PLAYPAUSE:
			if m.mediaHandler != nil && !m.tracker.IsStoped() {
				m.mediaHandler.OnPlayPause(m.tracker.IsPlaying())
			}
		}

	case tracker.SeekBy:
		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)
		m.mediaHandler.OnSeek(m.tracker.Position())
	case tracker.SeekTo:
		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)
		m.mediaHandler.OnSeek(m.tracker.Position())
	case tracker.VolumeTo:
		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)
		m.mediaHandler.OnVolume()
	case mediaShuffleMsg:
		items := m.playlists.Items()
		if m.currentPlaylistIndex >= 0 && m.currentPlaylistIndex < len(items) {
			currentPlaylist := items[m.currentPlaylistIndex]
			if len(currentPlaylist.Tracks) > 0 && currentPlaylist.Kind >= playlist.LIKES {
				cmd = m.shufflePlaylist(currentPlaylist)
				cmds = append(cmds, cmd)
			}
		}

	// search control update
	case search.Control:
		if m.isSearchActive {
			cmd = m.searchControl(msg)
			cmds = append(cmds, cmd)
		} else if m.isAddPlaylistActive {
			cmd = m.addPlaylistControl(msg)
			cmds = append(cmds, cmd)
		}

	// input dialog control update
	case input.Control:
		m.isRenamePlaylistActive = false
		cmd = m.renamePlaylistControl(msg)
		cmds = append(cmds, cmd)

	default:
		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)
		if m.isLoading {
			m.spinner, cmd = m.spinner.Update(message)
			cmds = append(cmds, cmd)
		} else if m.isSearchActive || m.isAddPlaylistActive {
			m.searchDialog, cmd = m.searchDialog.Update(message)
			cmds = append(cmds, cmd)
		} else if m.isRenamePlaylistActive {
			m.inputDialog, cmd = m.inputDialog.Update(message)
			cmds = append(cmds, cmd)
		} else {
			m.playlists, cmd = m.playlists.Update(message)
			cmds = append(cmds, cmd)
			m.tracklist, cmd = m.tracklist.Update(message)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.uiFocused && m.idleView != "" {
		return m.idleView
	}
	var view string
	switch {
	case m.isLoading:
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.spinner.View())
	case m.isSearchActive || m.isAddPlaylistActive:
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.searchDialog.View())
	case m.isRenamePlaylistActive:
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.inputDialog.View())
	default:
		playlistView := m.playlists.View()
		playlistWidth := lipgloss.Width(playlistView)

		m.tracker.SetWidth(m.width - playlistWidth - 2)
		m.tracklist.SetWidth(m.width - playlistWidth - 2)

		trackerView := m.tracker.View()
		trackerHeight := lipgloss.Height(trackerView)
		m.tracklist.SetHeight(m.height - trackerHeight - 2)

		tracklistView := m.tracklist.View()

		var midPanel string
		if m.tracklist.Hidden {
			midPanel = trackerView
		} else if m.tracker.Hidden {
			midPanel = tracklistView
		} else {
			midPanel = lipgloss.JoinVertical(lipgloss.Left, tracklistView, trackerView)
		}
		view = lipgloss.JoinHorizontal(lipgloss.Bottom, playlistView, midPanel)
	}

	if !m.uiFocused {
		view = style.Dimmed(view)
		m.idleView = view
		return view
	}
	return view
}

//
// private methods
//

func (m *Model) resize(width, height int) {
	m.width, m.height = width, height

	m.playlists.SetSize(style.SidePanelWidth, height-4)
	if !m.isPlaylistHideOverride {
		m.playlists.Hidden = m.width < style.SidePanelAutohide
	}

	searchWidth := style.SearchModalWidth
	if searchWidth > m.width {
		searchWidth = m.width - 2
	}

	m.searchDialog.SetSize(searchWidth, m.height-4)
	m.inputDialog.SetWidth(searchWidth)
}

func mediaPlaybackMsg(msg handler.Message) tea.Msg {
	switch msg.Type {
	case handler.MSG_NEXT:
		return tracker.NEXT
	case handler.MSG_PREVIOUS:
		return tracker.PREV
	case handler.MSG_PLAY:
		return tracker.PLAY
	case handler.MSG_PAUSE:
		return tracker.PAUSE
	case handler.MSG_PLAYPAUSE:
		return tracker.PLAYPAUSE
	case handler.MSG_STOP:
		return tracker.STOP
	case handler.MSG_SEEK:
		offset, ok := msg.Arg.(time.Duration)
		if !ok {
			return nil
		}
		return tracker.SeekBy(offset)
	case handler.MSG_SETPOS:
		pos, ok := msg.Arg.(time.Duration)
		if !ok {
			return nil
		}
		return tracker.SeekTo(pos)
	case handler.MSG_SET_VOLUME:
		vol, ok := msg.Arg.(float64)
		if !ok {
			return nil
		}
		return tracker.VolumeTo(vol)
	case handler.MSG_SET_SHUFFLE:
		val, ok := msg.Arg.(bool)
		if !ok || !val {
			return nil
		}
		return mediaShuffleMsg{}
	default:
		return nil
	}
}

func (m *Model) mediaHandle() {
	for msg := range m.mediaHandler.Message() {
		if playMsg := mediaPlaybackMsg(msg); playMsg != nil {
			m.Send(playMsg)
			continue
		}
		switch msg.Type {
		case handler.MSG_GET_PLAYBACKSTATUS, handler.MSG_GET_SHUFFLE, handler.MSG_GET_METADATA, handler.MSG_GET_VOLUME, handler.MSG_GET_POSITION:
			m.Send(msg)
		}
	}
}

func (m *Model) answerMediaGet(msg handler.Message) {
	if m.mediaHandler == nil {
		return
	}
	switch msg.Type {
	case handler.MSG_GET_PLAYBACKSTATUS:
		var state handler.PlaybackState
		if m.tracker.IsPlaying() {
			state = handler.STATE_PLAYING
		} else if m.tracker.IsStoped() {
			state = handler.STATE_STOPED
		} else {
			state = handler.STATE_PAUSED
		}
		m.mediaHandler.SendAnswer(state)
	case handler.MSG_GET_SHUFFLE:
		m.mediaHandler.SendAnswer(false)
	case handler.MSG_GET_METADATA:
		m.mediaHandler.SendAnswer(m.trackMetadata())
	case handler.MSG_GET_VOLUME:
		m.mediaHandler.SendAnswer(m.tracker.Volume())
	case handler.MSG_GET_POSITION:
		m.mediaHandler.SendAnswer(m.tracker.Position())
	}
}

func (m *Model) trackMetadata() handler.TrackMetadata {
	if m.tracker == nil || m.tracker.IsStoped() {
		return handler.TrackMetadata{}
	}
	return m.trackMetadataFrom(m.tracker.CurrentTrack())
}

func (m *Model) trackMetadataFrom(track *api.Track) handler.TrackMetadata {
	if track == nil {
		return handler.TrackMetadata{}
	}
	artists := make([]string, 0, len(track.Artists))
	for i := range track.Artists {
		artists = append(artists, track.Artists[i].Name)
	}
	albumArtists := make([]string, 0)
	var albumName string
	genre := make([]string, 0)
	if len(track.Albums) != 0 {
		for i := range track.Albums[0].Artists {
			albumArtists = append(albumArtists, track.Albums[0].Artists[i].Name)
		}
		albumName = track.Albums[0].Title
		genre = append(genre, track.Albums[0].Genre)
	}
	return handler.TrackMetadata{
		TrackId:      track.Id,
		Length:       time.Duration(track.DurationMs) * time.Millisecond,
		CoverUrl:     m.coverFilePath(),
		AlbumName:    albumName,
		AlbumArtists: albumArtists,
		Artists:      artists,
		Genre:        genre,
		Title:        track.Title,
		Url:          api.ShareTrackLink(track),
	}
}

func openOwnerFile(path string, flag int) (*os.File, error) {
	file, err := os.OpenFile(path, flag, 0600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func (m *Model) tempMediaDir() string {
	tempDir := filepath.Join(os.TempDir(), config.DirName)
	if os.MkdirAll(tempDir, 0700) != nil {
		return ""
	}
	if os.Chmod(tempDir, 0700) != nil {
		return ""
	}
	return tempDir
}

func (m *Model) coverFilePath() string {
	tempDir := m.tempMediaDir()
	if tempDir == "" {
		return ""
	}
	return filepath.Join(tempDir, "cover.jpg")
}

func (m *Model) removeCoverFiles() {
	tempDir := filepath.Join(os.TempDir(), config.DirName)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") {
			os.Remove(filepath.Join(tempDir, e.Name()))
		}
	}
}

func (m *Model) metadataFilePath() string {
	tempDir := m.tempMediaDir()
	if tempDir == "" {
		return ""
	}
	return filepath.Join(tempDir, "metadata.mp3")
}
