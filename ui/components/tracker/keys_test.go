package tracker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/config"
)

func TestVolumeKeysAreBrackets(t *testing.T) {
	config.InitTest()
	m := NewForTest()
	m.volume = 0.5
	m.volumeIncremet = 0.1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if cmd == nil {
		t.Fatal("] must change volume")
	}
	if m.volume <= 0.5 {
		t.Fatalf("] must raise volume, got %f", m.volume)
	}

	before := m.volume
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if cmd == nil {
		t.Fatal("[ must change volume")
	}
	if m.volume >= before {
		t.Fatalf("[ must lower volume, got %f", m.volume)
	}
}

func TestPlusDoesNotChangeVolume(t *testing.T) {
	config.InitTest()
	m := NewForTest()
	m.volume = 0.5
	before := m.volume
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if m.volume != before {
		t.Fatal("+ must not change volume")
	}
}
