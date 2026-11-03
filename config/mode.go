package config

import "strings"

const (
	PlayerModeAlwaysOn   = "always-on"
	PlayerModeWhenActive = "when-active"
)

func NormalizePlayerMode(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, " ", "-")
	switch s {
	case PlayerModeAlwaysOn, "alwayson":
		return PlayerModeAlwaysOn
	case PlayerModeWhenActive, "whenactive":
		return PlayerModeWhenActive
	default:
		return PlayerModeWhenActive
	}
}

func IsWhenActive() bool {
	return NormalizePlayerMode(Current.PlayerMode) == PlayerModeWhenActive
}

func TogglePlayerMode() string {
	if IsWhenActive() {
		Current.PlayerMode = PlayerModeAlwaysOn
	} else {
		Current.PlayerMode = PlayerModeWhenActive
	}
	return Current.PlayerMode
}
