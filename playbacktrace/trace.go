package playbacktrace

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dece2183/yamusic-tui/log"
)

const Enabled = false

const maxEvents = 2000

var (
	mu     sync.Mutex
	t0     time.Time
	track  string
	title  string
	events []string
)

func Begin(id, trackTitle string) {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	flushLocked("begin")
	t0 = time.Now()
	track = id
	title = trackTitle
	events = events[:0]
	appendLocked("begin id=%s title=%s", id, trackTitle)
}

func Event(format string, args ...any) {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if t0.IsZero() {
		t0 = time.Now()
	}
	appendLocked(format, args...)
}

func Flush(reason string) {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	flushLocked(reason)
}

func Snapshot() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(events))
	copy(out, events)
	return out
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	t0 = time.Time{}
	track = ""
	title = ""
	events = nil
}

func appendLocked(format string, args ...any) {
	if len(events) >= maxEvents {
		events = events[maxEvents/4:]
	}
	events = append(events, fmt.Sprintf("+%.3f %s", time.Since(t0).Seconds(), fmt.Sprintf(format, args...)))
}

func flushLocked(reason string) {
	if len(events) == 0 {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "playbacktrace reason=%s id=%s title=%s n=%d\n", reason, track, title, len(events))
	for _, line := range events {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("playbacktrace end\n")
	log.Dump(b.String())
	events = events[:0]
	t0 = time.Time{}
}
