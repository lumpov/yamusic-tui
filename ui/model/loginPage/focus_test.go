package loginpage

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/style"
	"github.com/muesli/termenv"
)

func TestLoginFocusToggle(t *testing.T) {
	config.InitTest()
	lipgloss.SetColorProfile(termenv.TrueColor)
	style.Apply(config.Current.Style)

	m := New()
	m.width, m.height = 40, 12
	m.uiFocused = true
	next, _ := m.Update(tea.BlurMsg{})
	got := next.(*Model)
	if got.uiFocused {
		t.Fatal("login BlurMsg must clear focus")
	}

	blurred := got.View()
	next, _ = got.Update(tea.FocusMsg{})
	got = next.(*Model)
	if !got.uiFocused {
		t.Fatal("login FocusMsg must set focus")
	}
	focused := got.View()
	if blurred == focused {
		t.Fatal("login view must change between blur and focus")
	}
}
