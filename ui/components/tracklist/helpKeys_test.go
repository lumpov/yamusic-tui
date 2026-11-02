package tracklist

import (
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestShortHelpOmitsShowKeys(t *testing.T) {
	config.InitTest(t)
	k := newHelpMap()
	k.HasArtist = true
	for _, b := range k.ShortHelp() {
		if b.Help().Desc == "show keys" {
			t.Fatal("tracklist short help must not include show keys")
		}
	}
}
