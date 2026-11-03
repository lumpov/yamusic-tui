package tracker

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/idleprevent"
	"github.com/dece2183/yamusic-tui/powerwatch"
	"github.com/dece2183/yamusic-tui/stream"
)

func playingModel(t *testing.T) (*Model, *fakePlayer) {
	t.Helper()
	config.InitTest(t)
	config.Current.PlayerMode = config.PlayerModeWhenActive
	m := NewForTest()
	fp := m.audio.(*fakeBackend).player
	fp.playing = true
	fp.volume = 0.5
	m.player = fp
	m.paused = false
	m.volume = 0.5
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(make([]byte, 8))), 8)
	t.Cleanup(func() { buf.Close() })
	m.trackWrapper = &readWrapper{trackBuffer: buf}
	return m, fp
}

func TestMCyclesPlayerMode(t *testing.T) {
	config.InitTest(t)
	m := NewForTest()
	if !config.IsWhenActive() {
		t.Fatal("default must be when-active")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("m must save player-mode")
	}
	if config.Current.PlayerMode != config.PlayerModeAlwaysOn {
		t.Fatalf("after m: %q, want %s", config.Current.PlayerMode, config.PlayerModeAlwaysOn)
	}
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if config.Current.PlayerMode != config.PlayerModeWhenActive {
		t.Fatalf("second m: %q, want %s", config.Current.PlayerMode, config.PlayerModeWhenActive)
	}
}

func TestDisplayOffPausesWhenActive(t *testing.T) {
	m, fp := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	if !m.paused {
		t.Fatal("display-off must pause in when-active")
	}
	if !m.pausedByPower {
		t.Fatal("display-off must mark pausedByPower")
	}
	if !fp.playing {
		t.Fatal("audio must keep playing until fade-out finishes")
	}
}

func TestDisplayOffIgnoredAlwaysOn(t *testing.T) {
	m, _ := playingModel(t)
	config.Current.PlayerMode = config.PlayerModeAlwaysOn
	m.Update(powerwatch.DisplayOff)
	if m.paused {
		t.Fatal("always-on must ignore display-off")
	}
	if m.pausedByPower {
		t.Fatal("always-on must not mark pausedByPower")
	}
}

func TestUnlockResumesAfterPowerPause(t *testing.T) {
	m, _ := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	m.Update(powerwatch.SessionUnlocked)
	if m.paused {
		t.Fatal("unlock must resume after display-off")
	}
	if m.pausedByPower {
		t.Fatal("unlock must clear pausedByPower")
	}
}

func TestUnlockResumeWithoutPlayerClearsPowerPause(t *testing.T) {
	m, _ := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	m.player = nil
	m.Update(powerwatch.SessionUnlocked)
	if m.pausedByPower {
		t.Fatal("unlock must clear pausedByPower when player is nil")
	}
}

func TestUnlockDoesNotResumeUserPause(t *testing.T) {
	m, _ := playingModel(t)
	m.Pause()
	m.Update(powerwatch.SessionUnlocked)
	if !m.paused {
		t.Fatal("unlock must not resume a user pause")
	}
}

func TestDisplayOnResumesWhenUnlocked(t *testing.T) {
	if powerwatch.ScreenLocked() {
		t.Skip("session is locked")
	}
	m, _ := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	m.Update(powerwatch.DisplayOn)
	if m.paused {
		t.Fatal("display-on without lock must resume")
	}
}

func TestDisplayOnDoesNotResumeWhenLocked(t *testing.T) {
	m, _ := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	m.Update(powerwatch.SessionLocked)
	m.Update(powerwatch.DisplayOn)
	if !m.paused {
		t.Fatal("display-on on lock screen must wait for unlock")
	}
	m.Update(powerwatch.SessionUnlocked)
	if m.paused {
		t.Fatal("unlock after lock screen must resume")
	}
}

func TestAlwaysOnToggleResumesPowerPause(t *testing.T) {
	m, _ := playingModel(t)
	m.Update(powerwatch.DisplayOff)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if config.Current.PlayerMode != config.PlayerModeAlwaysOn {
		t.Fatal("m must switch to always-on")
	}
	if m.paused {
		t.Fatal("switching to always-on must resume")
	}
}

func TestPowerResumeFadeInSevenSeconds(t *testing.T) {
	m, fp := playingModel(t)
	config.Current.Volume = 1
	m.volume = 1
	fp.volume = 1
	m.Update(powerwatch.DisplayOff)
	for i := 0; i < 80 && fp.playing; i++ {
		m.volumeFadeTick()
	}
	if fp.playing {
		t.Fatal("fade-out must finish")
	}
	m.Update(powerwatch.SessionUnlocked)
	if fp.volume != 0 {
		t.Fatalf("fade-in must start at 0, got %v", fp.volume)
	}
	n := 0
	for n < 400 && fp.volume != m.volume {
		m.volumeFadeTick()
		n++
	}
	if fp.volume != m.volume {
		t.Fatalf("fade-in did not reach target, ticks=%d vol=%v", n, fp.volume)
	}
	got := time.Duration(n) * _VOLUME_FADE_PERIOD
	if got < 6*time.Second || got > 8*time.Second {
		t.Fatalf("fade-in duration=%s ticks=%d, want ~7s", got, n)
	}
}

func TestAlwaysOnPlayingAcquiresIdlePrevent(t *testing.T) {
	t.Cleanup(func() { idleprevent.Set(false) })
	m, _ := playingModel(t)
	m.syncIdlePrevent()
	if idleprevent.Held() {
		t.Fatal("when-active must not hold idleprevent")
	}
	config.Current.PlayerMode = config.PlayerModeAlwaysOn
	m.syncIdlePrevent()
	if !idleprevent.Held() {
		t.Fatal("always-on while playing must hold idleprevent")
	}
	m.Pause()
	if idleprevent.Held() {
		t.Fatal("pause must release idleprevent")
	}
}
