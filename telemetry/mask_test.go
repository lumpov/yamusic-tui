package telemetry

import (
	"strings"
	"testing"

	"github.com/dece2183/yamusic-tui/config"
	"github.com/getsentry/sentry-go"
)

func TestMaskSecret(t *testing.T) {
	if got := MaskSecret(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := MaskSecret("short"); got != "..." {
		t.Fatalf("short: %q", got)
	}
	in := "02kdsjhABCDEFGkjds8"
	got := MaskSecret(in)
	if got != "02kdsjh...kjds8" {
		t.Fatalf("got %q want 02kdsjh...kjds8", got)
	}
}

func TestRedactEventReplacesToken(t *testing.T) {
	token := "02kdsjhABCDEFGkjds8"
	prev := config.Current.Token
	config.Current.Token = token
	t.Cleanup(func() { config.Current.Token = prev })

	event := &sentry.Event{
		Message:   "OAuth " + token + " failed",
		Exception: []sentry.Exception{{Value: "token=" + token}},
	}
	got := redactEvent(event)
	if strings.Contains(got.Message, token) {
		t.Fatalf("message still has token: %q", got.Message)
	}
	if strings.Contains(got.Exception[0].Value, token) {
		t.Fatalf("exception still has token: %q", got.Exception[0].Value)
	}
	if !strings.Contains(got.Message, "02kdsjh...kjds8") {
		t.Fatalf("message not masked: %q", got.Message)
	}
}

func TestStartEmptyDSNLeavesSentryOff(t *testing.T) {
	config.Current.SentryDSN = ""
	Start()
	if started {
		t.Fatal("empty DSN must not start Sentry")
	}
}
