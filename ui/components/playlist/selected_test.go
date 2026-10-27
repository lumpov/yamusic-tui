package playlist

import (
	"testing"

	"github.com/dece2183/yamusic-tui/config"
)

func TestSelectedItemEmptyList(t *testing.T) {
	config.InitTest()
	m := New(nil, "YaMusic")
	_ = m.Reset()
	if got := m.SelectedItem(); got != nil {
		t.Fatalf("empty list SelectedItem=%v, want nil", got)
	}
}

func TestViewEmptyList(t *testing.T) {
	config.InitTest()
	m := New(nil, "YaMusic")
	_ = m.Reset()
	m.SetSize(40, 20)
	_ = m.View()
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
