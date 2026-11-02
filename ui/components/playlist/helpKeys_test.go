package playlist

import (
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestShortHelpHasShowKeysAfterUpDown(t *testing.T) {
	config.InitTest(t)
	got := newHelpMap().ShortHelp()
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[2].Help().Desc != "show keys" {
		t.Fatalf("last desc=%q, want show keys", got[2].Help().Desc)
	}
}
