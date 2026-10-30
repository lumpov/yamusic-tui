package playlist

import (
	"strings"
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestSelectedItemEmptyList(t *testing.T) {
	config.InitTest(t)
	m := New(nil)
	_ = m.Reset()
	if got := m.SelectedItem(); got != nil {
		t.Fatalf("empty list SelectedItem=%v, want nil", got)
	}
}

func TestViewEmptyList(t *testing.T) {
	config.InitTest(t)
	m := New(nil)
	_ = m.Reset()
	m.SetSize(40, 20)
	_ = m.View()
}

func TestViewShowsTitleVersionAndUnofficialNote(t *testing.T) {
	config.InitTest(t)
	config.Version = "v0.9.0-beta.3"
	m := New(nil)
	_ = m.Reset()
	m.SetSize(40, 20)
	view := m.View()
	if !strings.Contains(view, config.AppTitle) {
		t.Fatalf("view missing title %q: %q", config.AppTitle, view)
	}
	if !strings.Contains(view, "v0.9.0-beta.3") {
		t.Fatalf("view missing version: %q", view)
	}
	if !strings.Contains(view, config.AppUnofficialNote) {
		t.Fatalf("view missing unofficial note: %q", view)
	}
}

func TestIsSameNil(t *testing.T) {
	var none *Item
	item := &Item{Name: "likes", Kind: LIKES}
	if none.IsSame(item) {
		t.Fatal("nil receiver must not match")
	}
	if item.IsSame(nil) {
		t.Fatal("nil argument must not match")
	}
	if !item.IsSame(&Item{Name: "likes", Kind: LIKES}) {
		t.Fatal("same name and kind must match")
	}
}
