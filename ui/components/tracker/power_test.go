package tracker

import (
	"bytes"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/config"
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
