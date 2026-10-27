package mainpage

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
)

func skipUIProgress(focused bool, msg tea.Msg) bool {
	if focused {
		return false
	}
	switch msg.(type) {
	case tracker.ProgressControl, progress.FrameMsg:
		return true
	default:
		return false
	}
}
