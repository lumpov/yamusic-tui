package api

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestTimeLimitedReaderDoesNotTimeoutBeforeFirstRead(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewTimeLimitedReader(pr, ctx, cancel, 40*time.Millisecond)
	defer r.Close()

	time.Sleep(80 * time.Millisecond)
	if ctx.Err() != nil {
		t.Fatal("timeout fired before first Read")
	}
}

func TestTimeLimitedReaderTimeoutsAfterIdleRead(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewTimeLimitedReader(pr, ctx, cancel, 40*time.Millisecond)
	defer r.Close()

	go func() {
		pw.Write([]byte("x"))
	}()

	buf := make([]byte, 1)
	if _, err := r.Read(buf); err != nil {
		t.Fatal(err)
	}

	time.Sleep(80 * time.Millisecond)
	if ctx.Err() == nil {
		t.Fatal("expected timeout after idle")
	}
}
