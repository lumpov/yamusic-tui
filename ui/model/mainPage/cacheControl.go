package mainpage

import (
	"os"

	"github.com/bogem/id3v2/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/cache"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/stream"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
)

type cacheReadyMsg struct {
	gen       int
	track     api.Track
	errBanner string
}

type cacheRemoveReadyMsg struct {
	gen       int
	trackId   string
	errBanner string
}

func (m *Model) cacheCurrentTrack() tea.Cmd {
	currentTrack := m.tracker.CurrentTrack()
	if m.tracker.IsStoped() || m.cachedTracksMap[currentTrack.Id] {
		return nil
	}

	buf := m.tracker.TrackBuffer()
	if buf == nil {
		return nil
	}
	trackCopy := *currentTrack
	metaPath := m.metadataFilePath()
	m.cacheGen++
	gen := m.cacheGen
	return func() tea.Msg {
		return writeCacheFile(gen, trackCopy, metaPath, buf)
	}
}

func writeCacheFile(gen int, track api.Track, metaPath string, buf *stream.BufferedStream) tea.Msg {
	metadataFile, err := openOwnerFile(metaPath, os.O_RDONLY)
	if err != nil {
		log.Print(log.LVL_ERROR, "failed to open cache file: %s", err)
		return cacheReadyMsg{gen: gen, errBanner: "cache open"}
	}
	defer metadataFile.Close()

	cacheFile, err := cache.Write(track.Id)
	if err != nil {
		log.Print(log.LVL_ERROR, "failed to write cache file: %s", err)
		return cacheReadyMsg{gen: gen, errBanner: "cache write"}
	}
	defer cacheFile.Close()

	tag := id3v2.NewEmptyTag()
	tag.Reset(metadataFile, id3v2.Options{Parse: true})
	tag.WriteTo(cacheFile)
	buf.WriteTo(cacheFile)
	return cacheReadyMsg{gen: gen, track: track}
}

func (m *Model) applyCache(msg cacheReadyMsg) tea.Cmd {
	if msg.gen != m.cacheGen {
		return nil
	}
	if msg.errBanner != "" {
		m.tracker.ShowError(msg.errBanner)
		return nil
	}

	m.cachedTracksMap[msg.track.Id] = true
	cachePlaylist, index := m.playlists.GetFirst(playlist.LOCAL)
	if cachePlaylist == nil || index < 0 {
		return nil
	}
	cachePlaylist.AddTrack(&msg.track)
	cmd := m.playlists.SetItem(index, cachePlaylist)

	if selected := m.playlists.SelectedItem(); selected != nil && selected.Kind == playlist.LOCAL {
		m.displayPlaylist(cachePlaylist)
	}

	m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	return cmd
}

func (m *Model) removeCache(track *api.Track) tea.Cmd {
	if track == nil {
		return nil
	}
	if m.tracker.CurrentTrack().Id == track.Id && len(m.tracker.CurrentTrack().RealId) == 0 {
		m.tracker.ShowError("can't remove currently playing track")
		return nil
	}

	trackId := track.Id
	m.cacheGen++
	gen := m.cacheGen
	return func() tea.Msg {
		err := cache.Remove(trackId)
		if err != nil {
			log.Print(log.LVL_ERROR, "failed to remove cached file: %s", err)
			return cacheRemoveReadyMsg{gen: gen, errBanner: "cache remove"}
		}
		return cacheRemoveReadyMsg{gen: gen, trackId: trackId}
	}
}

func (m *Model) applyCacheRemove(msg cacheRemoveReadyMsg) tea.Cmd {
	if msg.gen != m.cacheGen {
		return nil
	}
	if msg.errBanner != "" {
		m.tracker.ShowError(msg.errBanner)
		return nil
	}

	cachePlaylist, index := m.playlists.GetFirst(playlist.LOCAL)
	if cachePlaylist == nil || index < 0 {
		return nil
	}
	cachePlaylist.RemoveTrack(msg.trackId)

	delete(m.cachedTracksMap, msg.trackId)
	cmd := m.playlists.SetItem(index, cachePlaylist)

	if selected := m.playlists.SelectedItem(); selected != nil && selected.Kind == playlist.LOCAL {
		m.displayPlaylist(cachePlaylist)
	}

	m.indicateCurrentTrackPlaying(m.tracker.IsPlaying())
	return cmd
}
