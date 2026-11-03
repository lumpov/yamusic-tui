package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePlayerMode(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", PlayerModeWhenActive},
		{"nope", PlayerModeWhenActive},
		{"when-active", PlayerModeWhenActive},
		{"when active", PlayerModeWhenActive},
		{"WHEN_ACTIVE", PlayerModeWhenActive},
		{"always-on", PlayerModeAlwaysOn},
		{"always on", PlayerModeAlwaysOn},
		{"AlwaysOn", PlayerModeAlwaysOn},
	}
	for _, tc := range cases {
		if got := NormalizePlayerMode(tc.in); got != tc.want {
			t.Errorf("NormalizePlayerMode(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDefaultPlayerModeIsWhenActive(t *testing.T) {
	InitTest(t)
	if Current.PlayerMode != PlayerModeWhenActive {
		t.Fatalf("default PlayerMode=%q, want %q", Current.PlayerMode, PlayerModeWhenActive)
	}
	if !IsWhenActive() {
		t.Fatal("default must be when-active")
	}
}

func TestTogglePlayerMode(t *testing.T) {
	InitTest(t)
	if TogglePlayerMode() != PlayerModeAlwaysOn {
		t.Fatal("first toggle must be always-on")
	}
	if TogglePlayerMode() != PlayerModeWhenActive {
		t.Fatal("second toggle must be when-active")
	}
}

func TestPlayerModeFilledWhenMissing(t *testing.T) {
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
	if Current.PlayerMode != PlayerModeWhenActive {
		t.Fatalf("missing player-mode must default to %s, got %q", PlayerModeWhenActive, Current.PlayerMode)
	}
	if !Current.Controls.PlayerMode.Contains("m") {
		t.Fatal("missing player-mode key must default to m")
	}
}
