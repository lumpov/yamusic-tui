package tracklist

import (
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestSelectedItemEmptyList(t *testing.T) {
	config.InitTest(t)
	likes := make(map[string]bool)
	cache := make(map[string]bool)
	m := New(nil, &likes, &cache)
	if got := m.SelectedItem(); got.Track != nil || got.Album != nil {
		t.Fatalf("empty list SelectedItem=%+v, want zero item", got)
	}
}
