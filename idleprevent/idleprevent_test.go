package idleprevent

import (
	"runtime"
	"testing"
)

func TestSupportedMatchesGOOS(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "windows", "linux":
		if !Supported() {
			t.Fatalf("Supported()=false on %s", runtime.GOOS)
		}
	default:
		if Supported() {
			t.Fatalf("Supported()=true on %s", runtime.GOOS)
		}
	}
}

func TestSetIdempotent(t *testing.T) {
	Set(false)
	Set(true)
	Set(true)
	Set(false)
	Set(false)
	if Held() {
		t.Fatal("Set(false) must clear Held")
	}
}
