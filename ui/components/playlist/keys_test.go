package playlist

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/config"
)

func TestPlusMinusMovePlaylistCursor(t *testing.T) {
	config.InitTest()

	m := New(nil)
	_ = m.SetItems([]*Item{
		{Name: "one", Kind: LIKES, Active: true},
		{Name: "two", Kind: LIKES, Active: true},
	})
	m.Select(0)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	if cmd == nil {
		t.Fatal("minus must produce a command")
	}
	if m.Index() != 1 {
		t.Fatalf("minus: index=%d, want 1", m.Index())
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if cmd == nil {
		t.Fatal("plus must produce a command")
	}
	if m.Index() != 0 {
		t.Fatalf("plus: index=%d, want 0", m.Index())
	}
}

func TestCtrlUpIsNotPlaylistShortcut(t *testing.T) {
	config.InitTest()
	m := New(nil)
	_ = m.SetItems([]*Item{
		{Name: "one", Kind: LIKES, Active: true},
		{Name: "two", Kind: LIKES, Active: true},
	})
	m.Select(0)

	before := m.Index()
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp, Alt: false})
	if m.Index() != before {
		t.Fatal("plain up must not move playlists")
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if m.Index() != before {
		t.Fatal("ctrl+up must not move playlists")
	}
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if m.Index() != before {
		t.Fatal("ctrl+down must not move playlists")
	}
}
