package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultPlaylistKeysArePlusMinus(t *testing.T) {
	InitTest(t)

	if !Current.Controls.PlaylistsUp.Contains("+") {
		t.Fatalf("playlists-up: want +, keys=%v", Current.Controls.PlaylistsUp)
	}
	if Current.Controls.PlaylistsUp.Contains("ctrl+up") {
		t.Fatal("playlists-up still contains ctrl+up")
	}
	if !Current.Controls.PlaylistsDown.Contains("-") {
		t.Fatalf("playlists-down: want -, keys=%v", Current.Controls.PlaylistsDown)
	}
	if Current.Controls.PlaylistsDown.Contains("ctrl+down") {
		t.Fatal("playlists-down still contains ctrl+down")
	}
}

func TestDefaultVolumeKeysAreBrackets(t *testing.T) {
	InitTest(t)

	if !Current.Controls.PlayerVolUp.Contains("]") {
		t.Fatalf("player-vol-up: want ], keys=%v", Current.Controls.PlayerVolUp)
	}
	if Current.Controls.PlayerVolUp.Contains("+") {
		t.Fatal("player-vol-up still contains +")
	}
	if !Current.Controls.PlayerVolDown.Contains("[") {
		t.Fatalf("player-vol-down: want [, keys=%v", Current.Controls.PlayerVolDown)
	}
	if Current.Controls.PlayerVolDown.Contains("-") {
		t.Fatal("player-vol-down still contains -")
	}
}

func TestPlusIsNotVolumeAndNotCtrlUp(t *testing.T) {
	InitTest(t)

	if Current.Controls.PlaylistsUp.Contains("+") == Current.Controls.PlayerVolUp.Contains("+") {
		t.Fatal("+ must belong only to playlists-up")
	}
	if Current.Controls.PlaylistsDown.Contains("-") == Current.Controls.PlayerVolDown.Contains("-") {
		t.Fatal("- must belong only to playlists-down")
	}
}

func TestPlayerDislikeIsDAndUnique(t *testing.T) {
	InitTest(t)

	if Current.Controls.PlayerDislike == nil || !Current.Controls.PlayerDislike.Contains("d") {
		t.Fatalf("player-dislike: want d, keys=%v", Current.Controls.PlayerDislike)
	}
	if Current.Controls.PlayerDislike.Contains("D") {
		t.Fatal("player-dislike must not bind D")
	}
	if Current.Controls.PlayerDislike.Contains("down") {
		t.Fatal("player-dislike must not bind down")
	}

	owners := map[string]string{}
	v := reflect.ValueOf(Current.Controls).Elem()
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		k, ok := v.Field(i).Interface().(*Key)
		if !ok || k == nil {
			continue
		}
		for _, name := range k.keyNames {
			if other, exists := owners[name]; exists {
				t.Errorf("key %q belongs to %s and %s", name, other, typ.Field(i).Name)
			}
			owners[name] = typ.Field(i).Name
		}
	}
	if owners["d"] != "PlayerDislike" {
		t.Fatalf("d owner=%q, want PlayerDislike", owners["d"])
	}
	if owners["down"] != "CursorDown" {
		t.Fatalf("down owner=%q, want CursorDown", owners["down"])
	}
	if _, ok := owners["D"]; ok {
		t.Fatal("D must stay unbound")
	}
}

func TestPlayerDislikeFilledWhenMissing(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".config", DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := []byte("token: t\ncontrols:\n    player-like: L\n")
	if err := os.WriteFile(configPath(home), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitialLoad(); err != nil {
		t.Fatal(err)
	}
	if !Current.Controls.PlayerDislike.Contains("d") {
		t.Fatal("missing player-dislike must default to d")
	}
	if !Current.Controls.PlayerLike.Contains("L") {
		t.Fatal("player-like must stay L")
	}
}
