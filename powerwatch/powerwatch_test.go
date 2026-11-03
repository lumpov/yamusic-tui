package powerwatch

import (
	"runtime"
	"testing"
)

func TestSupportedMatchesGOOS(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "windows":
		if !Supported() {
			t.Fatalf("Supported()=false on %s", runtime.GOOS)
		}
	default:
		if Supported() {
			t.Fatalf("Supported()=true on %s", runtime.GOOS)
		}
	}
}

func TestEventValues(t *testing.T) {
	if DisplayOff == DisplayOn || SessionLocked == SessionUnlocked {
		t.Fatal("power events must be distinct")
	}
	if DisplayOff.String() != "display-off" || DisplayOn.String() != "display-on" ||
		SessionLocked.String() != "session-locked" || SessionUnlocked.String() != "session-unlocked" {
		t.Fatalf("event names: %s %s %s %s", DisplayOff, DisplayOn, SessionLocked, SessionUnlocked)
	}
}
