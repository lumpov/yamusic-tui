package style

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/muesli/termenv"
)

func TestDimmedStripsAccentAndAppliesInactiveColor(t *testing.T) {
	config.InitTest()
	lipgloss.SetColorProfile(termenv.TrueColor)
	Apply(config.Current.Style)

	colored := AccentTextStyle.Render("Hello")
	dim := Dimmed(colored)

	if dim == colored {
		t.Fatal("Dimmed must change the view")
	}
	if strings.Contains(dim, "Hello") == false {
		t.Fatal("Dimmed must keep the text")
	}
	if strings.Contains(dim, string(AccentColor)) {
		t.Fatal("Dimmed must strip accent color sequences")
	}
}
