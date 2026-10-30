package api

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/dece2183/yamusic-tui/playbacktrace"
)

type TimeLimitedReader struct {
	mu      sync.Mutex
	timer   *time.Timer
	timeout time.Duration
	cancel  context.CancelFunc
	ctx     context.Context
	body    io.ReadCloser
}

func NewTimeLimitedReader(r io.ReadCloser, ctx context.Context, ctxCancel context.CancelFunc, timeout time.Duration) *TimeLimitedReader {
	return &TimeLimitedReader{
		timeout: timeout,
		cancel:  ctxCancel,
		ctx:     ctx,
		body:    r,
	}
}

func (r *TimeLimitedReader) Read(dest []byte) (int, error) {
	r.mu.Lock()
	if r.timer == nil {
		r.timer = time.AfterFunc(r.timeout, func() {
			playbacktrace.Event("http idle-cancel timeout=%s", r.timeout)
			r.cancel()
		})
	} else {
		r.timer.Stop()
		r.timer.Reset(r.timeout)
	}
	r.mu.Unlock()
	t0 := time.Now()
	n, err := r.body.Read(dest)
	if d := time.Since(t0); d > 400*time.Millisecond {
		playbacktrace.Event("http read wait_ms=%d n=%d err=%v", d.Milliseconds(), n, err)
	}
	return n, err
}

func (r *TimeLimitedReader) Close() error {
	r.mu.Lock()
	if r.timer != nil {
		r.timer.Stop()
	}
	r.mu.Unlock()
	r.cancel()
	return r.body.Close()
}
