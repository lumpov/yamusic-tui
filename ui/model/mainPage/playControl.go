package mainpage

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	_ "image/jpeg"
	_ "image/png"

	"github.com/bogem/id3v2/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/cache"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/playbacktrace"
	"github.com/dece2183/yamusic-tui/stream"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
	"github.com/dece2183/yamusic-tui/ui/helpers"
)

const (
	_TRACK_DOWNLOAD_TRIES     = 3
	_TRACK_FINISHED_THRESHOLD = 0.8
)

func (m *Model) feedbackOnTrack(batch string) *api.RotorFeedback {
	currTrack := m.tracker.CurrentTrack()
	if currTrack == nil {
		return nil
	}
	var evType api.TrackEventType
	if m.tracker.Progress() > _TRACK_FINISHED_THRESHOLD {
		evType = api.EV_TRACK_FINISHED
	} else {
		evType = api.EV_TRACK_SKIPED
	}
	ev := api.NewTrackFeedbackEvent(evType, currTrack, m.tracker.Playtime().Seconds())
	fb := api.NewFeedback(batch, ev)
	log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type+" track: "+currTrack.Title)
	return fb
}

type rotateTracksReadyMsg struct {
	sessionId string
	batchId   string
	track     api.Track
	err       error
}

func (m *Model) rotateTracksCmd(pl *playlist.Item) tea.Cmd {
	if pl == nil || !pl.Rotor || m.client == nil {
		return nil
	}
	sessionId := pl.SessionId
	tracks := append([]api.Track(nil), pl.Tracks...)
	client := m.client
	return func() tea.Msg {
		suggestedTracks, err := client.RotorSessionTracks(sessionId, []*api.RotorFeedback{}, tracks)
		if err != nil {
			log.Print(log.LVL_ERROR, "failed to obtain more rotor tracks: %s", err)
			return rotateTracksReadyMsg{sessionId: sessionId, err: err}
		}
		if len(suggestedTracks.Sequence) == 0 {
			log.Print(log.LVL_ERROR, "failed to obtain more rotor tracks: empty sequence")
			return rotateTracksReadyMsg{sessionId: sessionId, err: fmt.Errorf("empty sequence")}
		}
		return rotateTracksReadyMsg{
			sessionId: sessionId,
			batchId:   suggestedTracks.BatchId,
			track:     suggestedTracks.Sequence[0].Track,
		}
	}
}

func (m *Model) applyRotateTracks(msg rotateTracksReadyMsg) {
	if msg.err != nil || msg.sessionId == "" {
		m.tracker.ShowError("next track obtain failure")
		return
	}
	items := m.playlists.Items()
	for i, pl := range items {
		if pl == nil || pl.SessionId != msg.sessionId {
			continue
		}
		pl.SessionBatch = msg.batchId
		pl.Tracks = append(pl.Tracks, msg.track)
		if m.currentPlaylistIndex >= 0 && m.currentPlaylistIndex < len(items) && items[m.currentPlaylistIndex].IsSame(pl) {
			m.playQueue = pl.Tracks
		}
		if selected := m.playlists.SelectedItem(); selected != nil && selected.IsSame(pl) {
			tackItems := m.tracklist.Items()
			if len(tackItems) > 0 {
				lastTrack := tackItems[len(tackItems)-1]
				lastTrack.IsSuggestion = false
				m.tracklist.SetItem(len(tackItems)-1, lastTrack)
			}
			m.tracklist.InsertItem(-1, tracklist.Item{
				Track:        &pl.Tracks[len(pl.Tracks)-1],
				Artists:      helpers.ArtistList(msg.track.Artists),
				IsSuggestion: true,
			})
		}
		m.playlists.SetItem(i, pl)
		return
	}
}

func (m *Model) prevTrack() tea.Cmd {
	if len(m.playQueue) == 0 || m.playQueueTrackIndex == 0 {
		m.Send(tracker.STOP)
		return nil
	}

	m.playQueueTrackIndex--
	for m.playQueueTrackIndex > 0 && !m.playQueue[m.playQueueTrackIndex].Available {
		m.playQueueTrackIndex--
	}

	if m.currentPlaylistIndex >= 0 {
		m.indicateCurrentTrackPlaying(false)
		currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
		if currentPlaylist.Rotor && m.tracker.IsPlaying() {
			go m.client.RotorSessionFeedback(currentPlaylist.SessionId, m.feedbackOnTrack(currentPlaylist.SessionBatch))
		}

		selectedPlaylist := m.playlists.SelectedItem()
		shouldFollow := currentPlaylist.IsSame(selectedPlaylist) && m.tracklist.Index() == currentPlaylist.CurrentTrack
		currentPlaylist.CurrentTrack = m.playQueueTrackIndex
		if shouldFollow {
			m.tracklist.Select(currentPlaylist.CurrentTrack)
			currentPlaylist.SelectedTrack = currentPlaylist.CurrentTrack
		}

		m.playlists.SetItem(m.currentPlaylistIndex, currentPlaylist)
	}

	track := &m.playQueue[m.playQueueTrackIndex]
	if !track.Available {
		m.Send(tracker.STOP)
		return nil
	}

	return m.playTrack(track)
}

func (m *Model) nextTrack() tea.Cmd {
	if len(m.playQueue) == 0 {
		m.Send(tracker.STOP)
		return nil
	}

	var rotate tea.Cmd
	m.playQueueTrackIndex++
	for m.playQueueTrackIndex < len(m.playQueue)-1 && !m.playQueue[m.playQueueTrackIndex].Available {
		m.playQueueTrackIndex++
	}

	if m.currentPlaylistIndex >= 0 {
		m.indicateCurrentTrackPlaying(false)
		currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
		if currentPlaylist.Rotor && m.tracker.IsPlaying() {
			go m.client.RotorSessionFeedback(currentPlaylist.SessionId, m.feedbackOnTrack(currentPlaylist.SessionBatch))
		}

		selectedPlaylist := m.playlists.SelectedItem()
		shouldFollow := currentPlaylist.IsSame(selectedPlaylist) && m.tracklist.Index() == currentPlaylist.CurrentTrack
		currentPlaylist.CurrentTrack = m.playQueueTrackIndex
		if shouldFollow {
			m.tracklist.Select(currentPlaylist.CurrentTrack)
			currentPlaylist.SelectedTrack = currentPlaylist.CurrentTrack
		}

		if currentPlaylist.CurrentTrack == len(m.playQueue)-1 {
			rotate = m.rotateTracksCmd(currentPlaylist)
		}

		m.playlists.SetItem(m.currentPlaylistIndex, currentPlaylist)
	}

	if m.playQueueTrackIndex >= len(m.playQueue) {
		m.playQueueTrackIndex = 0
		m.Send(tracker.STOP)
		return rotate
	}

	track := &m.playQueue[m.playQueueTrackIndex]
	if !track.Available {
		m.Send(tracker.STOP)
		return rotate
	}

	return tea.Batch(rotate, m.playTrack(track))
}

type playTrackReadyMsg struct {
	gen       int
	track     *api.Track
	buffer    *stream.BufferedStream
	lyrics    []api.LyricPair
	fromCache bool
	err       error
}

func (m *Model) playTrack(track *api.Track) tea.Cmd {
	m.tracker.Stop()
	m.playGen++
	gen := m.playGen
	return func() tea.Msg {
		return m.loadPlayTrack(gen, track)
	}
}

func (m *Model) loadPlayTrack(gen int, track *api.Track) playTrackReadyMsg {

	var (
		wg sync.WaitGroup

		coverType  string
		coverBytes []byte

		lyrics []api.LyricPair

		trackReader    io.ReadCloser
		trackSize      int64
		trackFromCache bool
		downloadErr    error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.removeCoverFiles()
		coverPath := m.coverFilePath()
		if coverPath == "" {
			return
		}
		coverFile, ferr := openOwnerFile(coverPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
		if ferr != nil {
			log.Print(log.LVL_WARNIGN, "unable to open cover file [%s]: %s", coverPath, ferr)
			return
		}
		keepCover := false
		defer func() {
			coverFile.Close()
			if !keepCover {
				os.Remove(coverPath)
			}
		}()
		cType, derr := api.DownloadTrackCover(coverFile, track, 200)
		if derr != nil {
			log.Print(log.LVL_WARNIGN, "unable to download track [%s] cover: %s", track.Id, derr)
			return
		}
		coverType = cType
		if serr := coverFile.Sync(); serr != nil {
			log.Print(log.LVL_WARNIGN, "unable to sync track [%s] cover: %s", track.Id, serr)
			return
		}
		if _, err := coverFile.Seek(0, io.SeekStart); err != nil {
			return
		}
		buf, rerr := io.ReadAll(coverFile)
		if rerr != nil {
			log.Print(log.LVL_WARNIGN, "unable to read track [%s] cover: %s", track.Id, rerr)
			return
		}
		coverBytes = buf
		keepCover = true
	}()

	if m.client != nil && track.Id != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lyr, lerr := m.client.TrackLyricsRequest(track.Id)
			if lerr != nil {
				log.Print(log.LVL_WARNIGN, "failed to obtain track [%s] lyrics: %s", track.Id, lerr)
				return
			}
			lyrics = lyr
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tr, ts, cerr := cache.Read(track.Id)
		if cerr == nil {
			trackReader = tr
			trackSize = ts
			trackFromCache = true
			return
		}
		var lastErr error
		for i := 0; i < _TRACK_DOWNLOAD_TRIES; i++ {
			trackInfos, ierr := m.client.TrackDownloadInfo(track.Id)
			if ierr != nil {
				log.Print(log.LVL_ERROR, "failed to obtain track [%s] info: %s", track.Id, ierr)
				lastErr = ierr
				continue
			}
			var bestBitrate int
			var bestTrackInfo api.TrackDownloadInfo
			for _, ti := range trackInfos {
				if ti.BbitrateInKbps > bestBitrate {
					bestBitrate = ti.BbitrateInKbps
					bestTrackInfo = ti
				}
			}
			tr2, ts2, derr := m.client.DownloadTrack(bestTrackInfo)
			if derr != nil {
				log.Print(log.LVL_ERROR, "failed to download track [%s]: %s", track.Id, derr)
				lastErr = derr
				continue
			}
			trackReader = tr2
			trackSize = ts2
			lastErr = nil
			break
		}
		downloadErr = lastErr
	}()

	wg.Wait()

	if downloadErr != nil && trackReader == nil {
		playbacktrace.Begin(track.Id, track.Title)
		playbacktrace.Event("load fail err=%v", downloadErr)
		playbacktrace.Flush("download-fail")
		return playTrackReadyMsg{gen: gen, track: track, err: downloadErr}
	}

	trackBuffer := stream.NewBufferedStream(trackReader, trackSize)
	metadataFile, err := openOwnerFile(m.metadataFilePath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY)
	if err == nil {
		tag := id3v2.NewEmptyTag()
		if trackFromCache {
			tag.Reset(trackBuffer, id3v2.Options{Parse: true})
		} else {
			tag.SetDefaultEncoding(id3v2.EncodingUTF8)
			tag.SetTitle(track.Title)
			if len(track.Albums) != 0 {
				tag.SetAlbum(track.Albums[0].Title)
				tag.SetGenre(track.Albums[0].Genre)
				tag.SetYear(fmt.Sprint(track.Albums[0].Year))
			}
			artists := make([]string, 0, len(track.Artists))
			for i := range track.Artists {
				artists = append(artists, track.Artists[i].Name)
				tag.AddFrame(cache.TAG_ARTIST_ID, id3v2.UserDefinedTextFrame{
					Encoding:    id3v2.EncodingUTF8,
					Description: fmt.Sprint(i),
					Value:       fmt.Sprint(track.Artists[i].Id),
				})
			}
			tag.SetArtist(strings.Join(artists, ","))
			tag.AddAttachedPicture(id3v2.PictureFrame{
				MimeType:    coverType,
				PictureType: id3v2.PTFrontCover,
				Encoding:    id3v2.EncodingUTF16BE,
				Picture:     coverBytes,
			})
			tag.AddTextFrame(cache.TAG_TRACK_ID, id3v2.EncodingUTF8, fmt.Sprint(track.Id))
			tag.AddTextFrame(cache.TAG_DURATION, id3v2.EncodingUTF8, fmt.Sprint(track.DurationMs))
		}
		tag.WriteTo(metadataFile)
		io.CopyN(metadataFile, trackBuffer, 32*1024)
		trackBuffer.Seek(0, io.SeekStart)
		metadataFile.Close()
	} else {
		log.Print(log.LVL_WARNIGN, "failed to create metadata file: %s", err)
	}

	return playTrackReadyMsg{
		gen:       gen,
		track:     track,
		buffer:    trackBuffer,
		lyrics:    lyrics,
		fromCache: trackFromCache,
	}
}

func (m *Model) applyPlayTrack(msg playTrackReadyMsg) {
	if msg.gen != m.playGen {
		if msg.buffer != nil {
			msg.buffer.Close()
		}
		return
	}
	if msg.err != nil || msg.buffer == nil {
		m.tracker.ShowError("track download")
		return
	}

	if m.currentPlaylistIndex >= 0 {
		currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
		if currentPlaylist.Rotor && m.client != nil {
			ev := api.NewTrackFeedbackEvent(api.EV_TRACK_STARTED, msg.track, 0)
			go m.client.RotorSessionFeedback(currentPlaylist.SessionId, api.NewFeedback(currentPlaylist.SessionBatch, ev))
			log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type+" track: "+msg.track.Title)
		}
	}

	m.tracker.StartTrack(msg.track, msg.buffer, msg.lyrics)
	playbacktrace.Event("load cache=%v size=%d", msg.fromCache, msg.buffer.Length())
	m.indicateCurrentTrackPlaying(true)
	if m.mediaHandler != nil {
		m.mediaHandler.OnPlayback()
	}

	if m.client != nil {
		go m.client.PlayTrack(msg.track, msg.fromCache)
	}
}

func (m *Model) playSelectedPlaylist(trackIndex int) tea.Cmd {
	selectedPlaylist := m.playlists.SelectedItem()
	if selectedPlaylist == nil || len(selectedPlaylist.Tracks) == 0 || selectedPlaylist.SelectedTrack < 0 || selectedPlaylist.SelectedTrack >= len(selectedPlaylist.Tracks) {
		m.Send(tracker.STOP)
		return nil
	}

	trackToPlay := &selectedPlaylist.Tracks[selectedPlaylist.SelectedTrack]

	if m.currentPlaylistIndex >= 0 {
		currentPlaylist := m.playlists.Items()[m.currentPlaylistIndex]
		if currentPlaylist.IsSame(selectedPlaylist) && selectedPlaylist.CurrentTrack == trackIndex && m.tracker.CurrentTrack().Id == trackToPlay.Id {
			if m.tracker.IsPlaying() {
				m.tracker.Pause()
				return nil
			} else {
				m.tracker.Play()
				return nil
			}
		}
		if currentPlaylist.Rotor {
			if m.tracker.IsPlaying() {
				go m.client.RotorSessionFeedback(currentPlaylist.SessionId, m.feedbackOnTrack(currentPlaylist.SessionBatch))
			}
			if !currentPlaylist.IsSame(selectedPlaylist) {
				ev := api.NewRadioFeedbackEvent(api.EV_RADIO_FINISHED)
				go m.client.RotorSessionFeedback(currentPlaylist.SessionId, api.NewFeedback(currentPlaylist.SessionBatch, ev))
				log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type)
			}
		}
	}

	m.indicateCurrentTrackPlaying(false)

	var rotate tea.Cmd
	if selectedPlaylist.Rotor {
		if m.currentPlaylistIndex != m.playlists.Index() {
			ev := api.NewRadioFeedbackEvent(api.EV_RADIO_STARTED)
			go m.client.RotorSessionFeedback(selectedPlaylist.SessionId, api.NewFeedback("", ev))
			log.Print(log.LVL_INFO, "feedback event sended: "+ev.Type)
			m.currentPlaylistIndex = m.playlists.Index()
		}
		if trackIndex == len(selectedPlaylist.Tracks)-1 {
			rotate = m.rotateTracksCmd(selectedPlaylist)
		}
	}

	if len(selectedPlaylist.Albums) > 0 && selectedPlaylist.SelectedAlbum >= 0 {
		m.currentAlbumIndex = selectedPlaylist.SelectedAlbum
	} else {
		m.currentAlbumIndex = -1
	}

	selectedPlaylist.CurrentTrack = trackIndex
	m.currentPlaylistIndex = m.playlists.Index()
	m.playlists.SetItem(m.currentPlaylistIndex, selectedPlaylist)
	m.playQueue = selectedPlaylist.Tracks
	m.playQueueTrackIndex = trackIndex
	return tea.Batch(rotate, m.playTrack(trackToPlay))
}
