package mainpage

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/style"
	"github.com/muesli/termenv"
)

func TestSkipUIProgressWhenBlurred(t *testing.T) {
	if !skipUIProgress(false, tracker.ProgressControl(0.4)) {
		t.Fatal("progress must be skipped while blurred")
	}
	if !skipUIProgress(false, progress.FrameMsg{}) {
		t.Fatal("progress frames must be skipped while blurred")
	}
	if skipUIProgress(true, tracker.ProgressControl(0.4)) {
		t.Fatal("progress must run while focused")
	}
	if skipUIProgress(false, tea.KeyMsg{}) {
		t.Fatal("keys must not be skipped while blurred")
	}
	if skipUIProgress(false, tracker.NEXT) {
		t.Fatal("playback controls must not be skipped while blurred")
	}
	if skipUIProgress(false, tracker.SeekBy(time.Second)) {
		t.Fatal("media seek must not be skipped while blurred")
	}
	if skipUIProgress(false, tracker.PLAYPAUSE) {
		t.Fatal("media playpause must not be skipped while blurred")
	}
}

func TestFocusAndBlurMessagesToggleFlag(t *testing.T) {
	config.InitTest()
	m := &Model{uiFocused: true}

	next, _ := m.Update(tea.BlurMsg{})
	got := next.(*Model)
	if got.uiFocused {
		t.Fatal("BlurMsg must set uiFocused=false")
	}

	next, _ = got.Update(tea.FocusMsg{})
	got = next.(*Model)
	if !got.uiFocused {
		t.Fatal("FocusMsg must set uiFocused=true")
	}
}

func TestViewDimsWhenBlurred(t *testing.T) {
	config.InitTest()
	lipgloss.SetColorProfile(termenv.TrueColor)
	style.Apply(config.Current.Style)

	m := &Model{
		uiFocused: true,
		isLoading: true,
		width:     80,
		height:    24,
		spinner:   spinner.New(spinner.WithSpinner(spinner.Points)),
	}

	focused := m.View()
	m.uiFocused = false
	blurred := m.View()

	if focused == blurred {
		t.Fatal("blurred view must differ from focused view")
	}
	if blurred != style.Dimmed(focused) {
		t.Fatal("blurred view must be the focused view passed through Dimmed")
	}
}
