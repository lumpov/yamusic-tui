package mainpage

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
)

// Канон: без фокуса окна UI не пересобирать и не анимировать.
// Bubble Tea после каждого Update всегда вызывает View (нет skip-render).
// skipUIProgress отбрасывает только paint-only сообщения (прогресс, кадры пружины, спиннер).
// Play/Pause/Seek/volumeFadeTick остаются в Update — плеер не останавливать.
// View при !uiFocused возвращает idleView (снимок Dimmed), без lipgloss по спискам.
func skipUIProgress(focused bool, msg tea.Msg) bool {
	if focused {
		return false
	}
	switch msg.(type) {
	case tracker.ProgressControl, progress.FrameMsg, spinner.TickMsg:
		return true
	default:
		return false
	}
}
