package config

import "testing"

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
