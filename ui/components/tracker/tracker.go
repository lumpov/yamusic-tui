package tracker

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/playbacktrace"
	"github.com/dece2183/yamusic-tui/stream"
	"github.com/dece2183/yamusic-tui/ui/helpers"
	"github.com/dece2183/yamusic-tui/ui/model"
	"github.com/dece2183/yamusic-tui/ui/style"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Control uint

const (
	PLAY Control = iota
	PAUSE
	STOP
	NEXT
	PREV
	LIKE
	REWIND
	VOLUME
	CACHE_TRACK
	BUFFERING_COMPLETE
	TOGGLE_LYRICS
	TOGGLE_VIEW
	PLAYPAUSE
)

type ProgressControl float64

func (p ProgressControl) Value() float64 {
	return float64(p)
}

const (
	_VOLUME_FADE_STEPS     = 2
	_VOLUME_SNAP_THRESHOLD = 0.005
	_VOLUME_FADE_PERIOD    = 60 * time.Millisecond
	_REWIND_SETTLE         = 200 * time.Millisecond
)

type SeekBy time.Duration
type SeekTo time.Duration
type VolumeTo float64

type rewindSettleMsg uint64
type volumeFadeTickMsg struct{}

type Model struct {
	width      int
	track      api.Track
	lyrics     []api.LyricPair
	progress   progress.Model
	volumeBar  progress.Model
	help       help.Model
	helpMap    *helpKeyMap
	Hidden     bool
	showLyrics bool
	showError  bool
	errorText  string

	paused            bool
	playtime          time.Duration
	playStarted       time.Time
	volume            float64
	volumeIncremet    float64
	lastVolumeKey     time.Time
	rewindAmount      time.Duration
	rewindBurst       bool
	rewindWasPlaying  bool
	rewindGen         uint64
	rewindTargetBytes int64
	audio             audioBackend
	player            audioPlayer
	trackWrapper      *readWrapper

	program  *tea.Program
	likesMap *map[string]bool
}

func New(p *tea.Program, likesMap *map[string]bool) *Model {
	m := &Model{
		program:    p,
		likesMap:   likesMap,
		progress:   progress.New(),
		volumeBar:  progress.New(),
		help:       help.New(),
		helpMap:    newHelpMap(),
		paused:     true,
		volume:     config.Current.Volume,
		showLyrics: config.Current.ShowLyrics,
	}

	m.volumeIncremet = m.volume / _VOLUME_FADE_STEPS
	m.rewindAmount = time.Duration(config.Current.RewindDuration) * time.Second

	m.progress.ShowPercentage = false
	m.progress.Empty = m.progress.Full
	m.progress.FullColor = string(style.AccentColor)
	m.progress.EmptyColor = string(style.BackgroundColor)
	m.progress.SetSpringOptions(60, 1)

	m.volumeBar.FullColor = string(style.AccentColor)
	m.volumeBar.EmptyColor = string(style.BackgroundColor)
	m.volumeBar.Width = style.VolumeIndicatorWidth

	m.help.Ellipsis = "…"
	m.trackWrapper = &readWrapper{program: m.program}
	m.audio = newAudioBackend()

	go m.volumeFader()
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) View() string {
	if m.Hidden {
		return ""
	}

	var playButton string
	if m.IsPlaying() {
		playButton = style.ActiveButtonStyle.Padding(0, 1).Margin(0).Render(style.IconPlay)
	} else {
		playButton = style.ActiveButtonStyle.Padding(0, 1).Margin(0).Render(style.IconStop)
	}

	var volumeIndicator string
	var volumeIndicatorWidth int
	if style.VolumeIndicatorWidth > 0 && m.width > style.VolumeIndicatorAutohide {
		var volumeIcon string
		if m.volume <= 0 {
			volumeIcon = style.IconVolumeOff
		} else if m.volume < 0.1 {
			volumeIcon = style.IconVolumeLow
		} else if m.volume < 0.5 {
			volumeIcon = style.IconVolumeMid
		} else {
			volumeIcon = style.IconVolumeHigh
		}
		volumeIndicator = " " + volumeIcon + " " + m.volumeBar.ViewAs(m.volume)
		volumeIndicatorWidth = lipgloss.Width(volumeIndicator)
	}

	m.progress.Width = m.width - volumeIndicatorWidth - 9
	tracker := style.TrackProgressStyle.Render(m.progress.ViewAs(m.progress.Percent()))
	tracker = lipgloss.JoinHorizontal(lipgloss.Top, playButton, tracker, volumeIndicator)

	if m.showLyrics {
		tracker = lipgloss.JoinVertical(lipgloss.Left, m.renderLyrics(), "", tracker)
	}

	if m.showError && !config.Current.SuppressErrors {
		errText := "Error: " + m.errorText + "; -> " + log.Location()
		maxLen := m.Width() - 4
		if lipgloss.Width(errText) > maxLen {
			errText = lipgloss.NewStyle().MaxWidth(maxLen-1).Render(errText) + "…"
		}
		tracker = lipgloss.JoinVertical(lipgloss.Left, style.ErrorTextStyle.Render(errText), "", tracker)
	}

	var trackTitle string
	if !m.help.ShowAll {
		if m.track.Available {
			trackTitle = style.TrackTitleStyle.Render(m.track.Title)
		} else {
			trackTitle = style.TrackTitleStyle.Strikethrough(true).Render(m.track.Title)
		}

		trackVersion := style.TrackVersionStyle.Render(" " + m.track.Version)
		trackTitle = lipgloss.JoinHorizontal(lipgloss.Top, trackTitle, trackVersion)

		durTotal := time.Millisecond * time.Duration(m.track.DurationMs)
		durEllapsed := time.Millisecond * time.Duration(float64(m.track.DurationMs)*m.progress.Percent())
		trackTime := style.TrackVersionStyle.Render(fmt.Sprintf("%02d:%02d/%02d:%02d",
			int(durEllapsed.Minutes()),
			int(durEllapsed.Seconds())%60,
			int(durTotal.Minutes()),
			int(durTotal.Seconds())%60,
		))

		var trackLike string
		if (*m.likesMap)[m.track.Id] {
			trackLike = style.IconLiked + " "
		} else {
			trackLike = style.IconNotLiked + " "
		}

		trackAddInfo := style.TrackAddInfoStyle.Render(trackLike + trackTime)
		addInfoLen := lipgloss.Width(trackAddInfo)
		maxLen := m.Width() - addInfoLen - 4
		stl := lipgloss.NewStyle().MaxWidth(maxLen - 1)

		trackTitleLen := lipgloss.Width(trackTitle)
		if trackTitleLen > maxLen {
			trackTitle = stl.Render(trackTitle) + "…"
		} else if trackTitleLen < maxLen {
			trackTitle += strings.Repeat(" ", maxLen-trackTitleLen)
		}

		trackArtist := style.TrackArtistStyle.Render(helpers.ArtistList(m.track.Artists))
		trackArtistLen := lipgloss.Width(trackArtist)
		if trackArtistLen > maxLen {
			trackArtist = stl.Render(trackArtist) + "…"
		} else if trackArtistLen < maxLen {
			trackArtist += strings.Repeat(" ", maxLen-trackArtistLen)
		}

		trackTitle = lipgloss.NewStyle().Width(m.width - lipgloss.Width(trackAddInfo) - 4).Render(trackTitle)
		trackTitle = lipgloss.JoinHorizontal(lipgloss.Top, trackTitle, trackAddInfo)
		trackTitle = lipgloss.JoinVertical(lipgloss.Left, trackTitle, trackArtist, "")

		tracker = lipgloss.JoinVertical(lipgloss.Left, tracker, trackTitle)
	}

	tracker = lipgloss.JoinVertical(lipgloss.Left, tracker, m.help.View(m.helpMap))
	return style.TrackBoxStyle.Width(m.width).Render(tracker)
}

func (m *Model) Update(message tea.Msg) (*Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := message.(type) {
	case tea.KeyMsg:
		controls := config.Current.Controls
		keypress := msg.String()

		switch {
		case controls.ShowAllKeys.Contains(keypress):
			m.help.ShowAll = !m.help.ShowAll

		case controls.PlayerPause.Contains(keypress):
			if m.player != nil && m.player.IsPlaying() {
				m.Pause()
				cmds = append(cmds, model.Cmd(PAUSE))
			} else {
				m.Play()
				cmds = append(cmds, model.Cmd(PLAY))
			}

		case controls.PlayerRewindForward.Contains(keypress):
			cmd = m.Rewind(m.rewindAmount)
			cmds = append(cmds, cmd, model.Cmd(REWIND))

		case controls.PlayerRewindBackward.Contains(keypress):
			cmd = m.Rewind(-m.rewindAmount)
			cmds = append(cmds, cmd, model.Cmd(REWIND))

		case controls.PlayerNext.Contains(keypress):
			cmds = append(cmds, model.Cmd(NEXT))

		case controls.PlayerPrevious.Contains(keypress):
			cmds = append(cmds, model.Cmd(PREV))

		case controls.PlayerLike.Contains(keypress):
			cmds = append(cmds, model.Cmd(LIKE))

		case controls.PlayerCache.Contains(keypress):
			if !m.IsStoped() {
				cmds = append(cmds, model.Cmd(CACHE_TRACK))
			}

		case controls.PlayerVolUp.Contains(keypress):
			m.SetVolume(m.volume + m.dynamicVolumeStep())
			cmds = append(cmds, model.Cmd(VOLUME))

		case controls.PlayerVolDown.Contains(keypress):
			m.SetVolume(m.volume - m.dynamicVolumeStep())
			cmds = append(cmds, model.Cmd(VOLUME))

		case controls.PlayerToggleLyrics.Contains(keypress):
			m.SetLirycs(!m.showLyrics)
			cmds = append(cmds, model.Cmd(TOGGLE_LYRICS))
		case controls.PlayerHide.Contains(keypress):
			m.Hidden = !m.Hidden
			cmds = append(cmds, model.Cmd(TOGGLE_VIEW))
		}

	// player control update
	case Control:
		switch msg {
		case PLAY:
			m.Play()
		case PAUSE:
			m.Pause()
		case STOP:
			m.Stop()
		case PLAYPAUSE:
			if m.player != nil && m.player.IsPlaying() {
				m.Pause()
			} else {
				m.Play()
			}
		}

	case SeekBy:
		cmd = m.Rewind(time.Duration(msg))
		cmds = append(cmds, cmd)

	case SeekTo:
		m.SetPos(time.Duration(msg))

	case VolumeTo:
		m.SetVolume(float64(msg))

	case volumeFadeTickMsg:
		m.volumeFadeTick()

	// track progress update
	case ProgressControl:
		cmd = m.progress.SetPercent(msg.Value())
		cmds = append(cmds, cmd)

	case rewindSettleMsg:
		if uint64(msg) == m.rewindGen {
			m.finishRewindBurst()
		}

	case progress.FrameMsg:
		var progressModel tea.Model
		progressModel, cmd = m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) SetWidth(width int) {
	m.width = width
	m.help.Width = width - 4
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	baseHeight := 4
	if m.showLyrics {
		baseHeight += 4
	}
	if m.showError && !config.Current.SuppressErrors {
		baseHeight += 2
	}
	return baseHeight
}

func (m *Model) Progress() float64 {
	return m.progress.Percent()
}

func (m *Model) Position() time.Duration {
	return time.Duration(float64(m.track.DurationMs)*m.trackWrapper.Progress()) * time.Millisecond
}

func (m *Model) SetVolume(v float64) {
	if v < _VOLUME_SNAP_THRESHOLD {
		v = 0
	} else if v > 1-_VOLUME_SNAP_THRESHOLD {
		v = 1
	}
	m.volume = v
	m.volumeIncremet = m.volume / _VOLUME_FADE_STEPS
	config.Current.Volume = m.volume
	config.Save()
}

func (m *Model) SetLirycs(show bool) {
	m.showLyrics = show
	config.Current.ShowLyrics = m.showLyrics
	config.Save()
}

func (m *Model) Volume() float64 {
	return m.volume
}

func (m *Model) StartTrack(track *api.Track, reader *stream.BufferedStream, lyrics []api.LyricPair) {
	m.showError = false
	m.volume = config.Current.Volume
	m.volumeIncremet = m.volume / _VOLUME_FADE_STEPS

	if m.player != nil {
		m.Stop()
	}

	m.track = *track
	if err := m.trackWrapper.NewReader(reader); err != nil {
		m.trackWrapper.Close()
		m.ShowError("mp3 decoder")
		return
	}
	synced := 0
	for _, line := range lyrics {
		if line.Timestamp > 0 {
			synced++
		}
	}
	playbacktrace.Begin(track.Id, track.Title)
	playbacktrace.Event("start dur_ms=%d lyrics=%d synced=%d len=%d", track.DurationMs, len(lyrics), synced, reader.Length())
	m.player = m.audio.Open(m.trackWrapper)
	m.player.SetVolume(0)
	m.player.Play()
	m.lyrics = lyrics
	m.paused = false
	m.playtime = 0
	m.playStarted = time.Now()
	m.rewindBurst = false
	m.rewindGen++
}

func (m *Model) Stop() {
	burst := m.rewindBurst
	m.rewindBurst = false
	m.rewindGen++
	if m.player == nil {
		return
	}

	if m.player.IsPlaying() {
		m.player.SetVolume(0)
		m.player.Pause()
	}

	bufErr := error(nil)
	bufProg := 0.0
	readProg := 0.0
	if m.trackWrapper != nil && m.trackWrapper.trackBuffer != nil {
		bufErr = m.trackWrapper.trackBuffer.Error()
		bufProg = m.trackWrapper.trackBuffer.BufferingProgress()
		readProg = m.trackWrapper.trackBuffer.Progress()
	}
	playbacktrace.Event("stop playing=%v paused=%v burst=%v read=%.3f buf=%.3f err=%v", m.player.IsPlaying(), m.paused, burst, readProg, bufProg, bufErr)

	if bufErr != nil {
		log.Print(log.LVL_ERROR, "track buffering: %s", bufErr)
		m.ShowError("track buffering")
	}

	m.trackWrapper.Close()
	m.player.Close()
	m.player = nil
	m.playtime += time.Since(m.playStarted)
	m.paused = true
	playbacktrace.Flush("stop")
}

func (m *Model) IsPlaying() bool {
	return m.player != nil && m.trackWrapper.trackBuffer != nil && m.player.IsPlaying() && !m.paused
}

func (m *Model) IsStoped() bool {
	return m.player == nil || m.trackWrapper.trackBuffer == nil
}

func (m *Model) CurrentTrack() *api.Track {
	return &m.track
}

func (m *Model) Play() {
	if m.player == nil || m.trackWrapper.trackBuffer == nil {
		return
	}
	if m.player.IsPlaying() {
		return
	}
	m.volume = config.Current.Volume
	m.volumeIncremet = m.volume / _VOLUME_FADE_STEPS
	m.player.SetVolume(0)
	m.player.Play()
	m.paused = false
	m.playStarted = time.Now()
	playbacktrace.Event("play")
}

func (m *Model) Pause() {
	if m.player == nil || m.trackWrapper.trackBuffer == nil {
		return
	}
	if !m.player.IsPlaying() {
		return
	}
	m.playtime += time.Since(m.playStarted)
	m.paused = true
	playbacktrace.Event("pause")
}

func (m *Model) Rewind(amount time.Duration) tea.Cmd {
	if m.player == nil || m.trackWrapper == nil {
		go m.program.Send(STOP)
		return nil
	}

	length := m.trackWrapper.Length()
	current := int64(float64(length) * m.trackWrapper.Progress())
	if m.rewindBurst {
		current = m.rewindTargetBytes
	} else {
		m.rewindWasPlaying = m.IsPlaying()
		m.rewindBurst = true
		m.paused = true
		m.player.SetVolume(0)
		m.player.Pause()
	}
	pos := current + compressedByteOffset(length, float64(m.track.DurationMs), float64(amount.Milliseconds()))
	pos = clampPlayableByteOffset(pos, length, m.rewindStepBytes())
	pos -= pos % 4
	m.rewindTargetBytes = pos
	playbacktrace.Event("rewind amount=%s burst=%v from=%d to=%d len=%d", amount, m.rewindBurst, current, pos, length)
	if _, err := m.player.Seek(pos, io.SeekStart); err != nil {
		log.Print(log.LVL_ERROR, "rewind seek: %s", err)
		playbacktrace.Event("rewind seek err=%v", err)
	}

	m.rewindGen++
	gen := m.rewindGen
	return tea.Batch(
		m.progress.SetPercent(m.trackWrapper.Progress()),
		tea.Tick(_REWIND_SETTLE, func(time.Time) tea.Msg {
			return rewindSettleMsg(gen)
		}),
	)
}

func (m *Model) rewindStepBytes() int64 {
	if m.trackWrapper == nil {
		return 0
	}
	step := compressedByteOffset(m.trackWrapper.Length(), float64(m.track.DurationMs), float64(m.rewindAmount.Milliseconds()))
	if step < 4 {
		return 4
	}
	return step
}

func (m *Model) finishRewindBurst() {
	if !m.rewindBurst {
		return
	}
	m.rewindBurst = false
	m.rewindTargetBytes = 0
	if m.player == nil {
		return
	}
	if m.rewindWasPlaying {
		m.paused = false
		m.playStarted = time.Now()
		m.player.SetVolume(m.volume)
		m.player.Play()
		playbacktrace.Event("rewind settle play isPlaying=%v", m.player.IsPlaying())
	} else {
		playbacktrace.Event("rewind settle stay-paused")
	}
}

func (m *Model) SetPos(pos time.Duration) {
	if m.player == nil || m.trackWrapper == nil {
		go m.program.Send(STOP)
		return
	}

	length := m.trackWrapper.Length()
	byteOffset := clampPlayableByteOffset(
		compressedByteOffset(length, float64(m.track.DurationMs), float64(pos.Milliseconds())),
		length,
		m.rewindStepBytes(),
	)
	byteOffset -= byteOffset % 4
	m.player.Seek(byteOffset, io.SeekStart)
}

func (m *Model) TrackBuffer() *stream.BufferedStream {
	return m.trackWrapper.trackBuffer
}

func (m *Model) Playtime() time.Duration {
	if m.paused {
		return m.playtime
	}
	return m.playtime + time.Since(m.playStarted)
}

func (m *Model) ShowError(text string) {
	m.showError = true
	m.errorText = text
}

func (m *Model) HideError() {
	m.showError = false
}

func (m *Model) dynamicVolumeStep() float64 {
	now := time.Now()
	delta := now.Sub(m.lastVolumeKey)
	m.lastVolumeKey = now

	step := config.Current.VolumeStep
	switch {
	case delta < 80*time.Millisecond:
		return step
	case delta < 200*time.Millisecond:
		return step * 0.4
	default:
		return step * 0.2
	}
}

func (m *Model) volumeFadeTick() {
	if m.player == nil || !m.player.IsPlaying() {
		return
	}

	if m.volumeIncremet == 0 {
		m.player.SetVolume(0)
		return
	}

	var targetVolume float64
	if m.paused {
		targetVolume = 0
	} else {
		targetVolume = m.volume
	}

	currVol := m.player.Volume()
	if currVol >= targetVolume+m.volumeIncremet {
		m.player.SetVolume(currVol - m.volumeIncremet/2)
	} else if currVol <= targetVolume-m.volumeIncremet {
		m.player.SetVolume(currVol + m.volumeIncremet/2)
	} else if currVol != targetVolume {
		m.player.SetVolume(targetVolume)
		if m.paused {
			m.player.Pause()
		}
	}
}

func (m *Model) needsVolumeFade() bool {
	if m.player == nil || !m.player.IsPlaying() {
		return false
	}
	if m.volumeIncremet == 0 {
		return m.player.Volume() != 0
	}
	target := m.volume
	if m.paused {
		target = 0
	}
	return m.player.Volume() != target
}

func (m *Model) volumeFader() {
	t := time.NewTicker(_VOLUME_FADE_PERIOD)
	for range t.C {
		if !m.needsVolumeFade() {
			continue
		}
		if m.program == nil {
			m.volumeFadeTick()
			continue
		}
		m.program.Send(volumeFadeTickMsg{})
	}
}

func (m *Model) renderLyrics() string {
	currentLine := " "
	nextLine := " "
	previousLine := " "

	if m.player != nil && m.showLyrics {
		if len(m.lyrics) == 0 {
			currentLine = "No lyrics for this track"
		} else if m.lyricsHaveTimestamps() {
			for idx, line := range m.lyrics {
				if line.Timestamp > int(m.Position().Milliseconds()-1000) {
					previousLine = m.tryGetLyricsLine(idx - 2)
					currentLine = m.lyricsBreak(m.tryGetLyricsLine(idx - 1))
					nextLine = m.tryGetLyricsLine(idx)
					break
				}
			}
		} else {
			idx := m.unsyncedLyricsIndex()
			previousLine = m.tryGetLyricsLine(idx - 1)
			currentLine = m.lyricsBreak(m.tryGetLyricsLine(idx))
			nextLine = m.tryGetLyricsLine(idx + 1)
		}
	}

	previousLine = lipgloss.NewStyle().Foreground(style.LyricsPreviosTextColor).Render(previousLine)
	nextLine = lipgloss.NewStyle().Foreground(style.LyricsNextTextColor).Render(nextLine)
	currentLine = lipgloss.NewStyle().Foreground(style.LyricsCurrentTextColor).Render(currentLine)

	lyrics := lipgloss.JoinVertical(lipgloss.Center, previousLine, currentLine, nextLine)
	lyrics = lipgloss.NewStyle().Width(m.width - 4).AlignHorizontal(lipgloss.Center).Render(lyrics)

	return lyrics
}

func (m *Model) lyricsHaveTimestamps() bool {
	for _, line := range m.lyrics {
		if line.Timestamp > 0 {
			return true
		}
	}
	return false
}

func (m *Model) unsyncedLyricsIndex() int {
	n := len(m.lyrics)
	if n == 0 {
		return 0
	}
	progress := 0.0
	if m.track.DurationMs > 0 && m.trackWrapper != nil {
		progress = float64(m.Position().Milliseconds()) / float64(m.track.DurationMs)
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	idx := int(progress * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return idx
}

func (m *Model) tryGetLyricsLine(idx int) (line string) {
	if idx < 0 || idx >= len(m.lyrics) {
		return
	}
	return m.lyrics[idx].Line
}

func (m *Model) lyricsBreak(line string) (newLine string) {
	if strings.TrimSpace(strings.TrimSpace(line)) != "" {
		return line
	}

	switch m.Position().Milliseconds() % 900 / 300 {
	default:
		newLine = style.IconDotLight + style.IconDotDark + style.IconDotDark
	case 1:
		newLine = style.IconDotDark + style.IconDotLight + style.IconDotDark
	case 2:
		newLine = style.IconDotDark + style.IconDotDark + style.IconDotLight
	}

	return
}
