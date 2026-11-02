package tracker

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/yamusic-tui/config"
)

func TestWrapShortHelpFitsWidthAndKeepsDislike(t *testing.T) {
	config.InitTest(t)
	h := help.New()
	h.Width = 40
	view := wrapShortHelp(h, newHelpMap().ShortHelp())
	if !strings.Contains(view, "dislike") {
		t.Fatal("wrapped help must keep dislike")
	}
	if !strings.Contains(view, "\n") {
		t.Fatal("width 40 must wrap tracker short help")
	}
	for _, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > 40 {
			t.Fatalf("line width %d > 40: %q", w, line)
		}
	}
}
