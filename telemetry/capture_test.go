package telemetry

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

type captureTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *captureTransport) Configure(sentry.ClientOptions) {}
func (t *captureTransport) Flush(time.Duration) bool       { return true }
func (t *captureTransport) Close()                         {}
func (t *captureTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *captureTransport) last() *sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.events) == 0 {
		return nil
	}
	return t.events[len(t.events)-1]
}

func withTestHub(t *testing.T, tr *captureTransport) {
	t.Helper()
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "http://whatever@example.com/1337",
		Transport: tr,
	})
	if err != nil {
		t.Fatal(err)
	}
	hub := sentry.CurrentHub()
	prev := hub.Client()
	hub.BindClient(client)
	started = true
	t.Cleanup(func() {
		started = false
		hub.BindClient(prev)
	})
}

func TestCaptureBufferingNoopWhenOff(t *testing.T) {
	started = false
	CaptureBuffering(errors.New("use of closed network connection"), nil, nil)
}

func TestAddBreadcrumbNoopWhenOff(t *testing.T) {
	started = false
	AddBreadcrumb("power", "display-off", nil)
}

func TestCaptureWhenActiveNoopWhenOff(t *testing.T) {
	started = false
	CaptureWhenActive("resume without player", nil)
}

func TestCaptureBufferingSendsTaggedException(t *testing.T) {
	tr := &captureTransport{}
	withTestHub(t, tr)
	AddBreadcrumb("power", "display-off", map[string]interface{}{"mode": "when-active"})
	CaptureBuffering(errors.New("use of closed network connection"), map[string]string{
		"player_mode":     "when-active",
		"paused_by_power": "true",
	}, map[string]interface{}{"buf_progress": 0.3})
	if !sentry.Flush(2 * time.Second) {
		t.Fatal("flush")
	}
	ev := tr.last()
	if ev == nil {
		t.Fatal("no event")
	}
	if ev.Tags["kind"] != "track-buffering" {
		t.Fatalf("kind=%q", ev.Tags["kind"])
	}
	if ev.Tags["player_mode"] != "when-active" {
		t.Fatalf("player_mode=%q", ev.Tags["player_mode"])
	}
	if ev.Tags["paused_by_power"] != "true" {
		t.Fatalf("paused_by_power=%q", ev.Tags["paused_by_power"])
	}
	if len(ev.Fingerprint) == 0 || ev.Fingerprint[0] != "track-buffering" {
		t.Fatalf("fingerprint=%v", ev.Fingerprint)
	}
	found := false
	for _, b := range ev.Breadcrumbs {
		if b != nil && b.Category == "power" && b.Message == "display-off" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing power breadcrumb: %+v", ev.Breadcrumbs)
	}
}

func TestCaptureWhenActiveSendsWarning(t *testing.T) {
	tr := &captureTransport{}
	withTestHub(t, tr)
	CaptureWhenActive("resume without player", map[string]string{"player_mode": "when-active"})
	if !sentry.Flush(2 * time.Second) {
		t.Fatal("flush")
	}
	ev := tr.last()
	if ev == nil {
		t.Fatal("no event")
	}
	if ev.Message != "when-active: resume without player" {
		t.Fatalf("message=%q", ev.Message)
	}
	if ev.Level != sentry.LevelWarning {
		t.Fatalf("level=%q", ev.Level)
	}
	if len(ev.Fingerprint) < 2 || ev.Fingerprint[0] != "when-active" {
		t.Fatalf("fingerprint=%v", ev.Fingerprint)
	}
}
