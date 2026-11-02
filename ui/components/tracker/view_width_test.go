package tracker

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/style"
)

func TestTrackerViewFitsTerminalWidth(t *testing.T) {
	config.InitTest(t)
	style.Apply(config.Current.Style)

	likes := map[string]bool{"1": true}
	dislikes := map[string]bool{"1": true}
	m := NewForTest()
	m.likesMap = &likes
	m.SetDislikesMap(&dislikes)
	m.track = api.Track{
		Id:         "1",
		Title:      "Very long track title that must be clipped inside the player box",
		Version:    "remix edition deluxe",
		Available:  true,
		DurationMs: 215000,
		Artists:    []api.Artist{{Name: "A very long artist name and another one"}},
	}

	for _, width := range []int{24, 40, 64, 80, 120} {
		m.SetWidth(width)
		view := m.View()
		got := 0
		for _, line := range strings.Split(view, "\n") {
			if w := lipgloss.Width(line); w > got {
				got = w
			}
		}
		if got > width {
			t.Fatalf("width=%d view line=%d, want <= %d\n%s", width, got, width, view)
		}
	}
}
