package playlist

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/style"
)

func TestViewModeBetweenUnofficialAndMyWave(t *testing.T) {
	config.InitTest(t)
	style.Apply(config.Current.Style)
	config.Current.PlayerMode = config.PlayerModeWhenActive
	m := New(nil)
	_ = m.SetItems([]*Item{{Name: "my wave", Kind: MYWAVE, Active: true}})
	m.SetSize(40, 20)
	plain := ansi.Strip(m.View())
	iNote := strings.Index(plain, config.AppUnofficialNote)
	iMode := strings.Index(plain, "mode:")
	iWave := strings.Index(plain, "my wave")
	if iNote < 0 || iMode < 0 || iWave < 0 {
		t.Fatalf("view missing unofficial/mode/my wave: %q", plain)
	}
	if !(iNote < iMode && iMode < iWave) {
		t.Fatalf("mode must be between unofficial and my wave: %q", plain)
	}
	if !strings.Contains(plain, "always on") || !strings.Contains(plain, "when active") {
		t.Fatalf("view must show both modes: %q", plain)
	}

	lines := strings.Split(plain, "\n")
	noteLine, modeLine, waveLine := -1, -1, -1
	for i, line := range lines {
		switch {
		case noteLine < 0 && strings.Contains(line, config.AppUnofficialNote):
			noteLine = i
		case modeLine < 0 && strings.Contains(line, "mode:"):
			modeLine = i
		case waveLine < 0 && strings.Contains(line, "my wave"):
			waveLine = i
		}
	}
	if noteLine < 0 || modeLine < 0 || waveLine < 0 {
		t.Fatalf("lines missing unofficial/mode/my wave: %q", plain)
	}
	if modeLine-noteLine < 2 || waveLine-modeLine < 2 {
		t.Fatalf("blank lines around mode missing: note=%d mode=%d wave=%d\n%s", noteLine, modeLine, waveLine, plain)
	}

	m.SetSize(32, 20)
	plain32 := ansi.Strip(m.View())
	if !strings.Contains(plain32, "mode:") || !strings.Contains(plain32, "when active") {
		t.Fatalf("width 32 must keep mode line: %q", plain32)
	}
}
