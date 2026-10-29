package stream

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

type closeRecorder struct {
	io.Reader
	mu     sync.Mutex
	closed int
}

func (c *closeRecorder) Close() error {
	c.mu.Lock()
	c.closed++
	c.mu.Unlock()
	return nil
}

func (c *closeRecorder) closeCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func waitBuffered(t *testing.T, s *BufferedStream, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mux.Lock()
		n := len(s.readBuffer)
		s.mux.Unlock()
		if n >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("buffer len never reached %d", want)
}

func TestBufferFramesKeepsUnexpectedEOFTail(t *testing.T) {
	data := bytes.Repeat([]byte("ab"), 50)
	src := &closeRecorder{Reader: bytes.NewReader(data)}
	s := NewBufferedStream(src, int64(len(data)))
	defer s.Close()
	waitBuffered(t, s, len(data))
	s.mux.Lock()
	got := append([]byte(nil), s.readBuffer...)
	s.mux.Unlock()
	if !bytes.Equal(got, data) {
		t.Fatalf("len=%d, want %d", len(got), len(data))
	}
}

func TestSeekEndCloseClosesSource(t *testing.T) {
	data := []byte("xyz")
	src := &closeRecorder{Reader: bytes.NewReader(data)}
	s := NewBufferedStream(src, int64(len(data)))
	if _, err := s.Seek(0, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if src.closeCount() != 1 {
		t.Fatalf("Close count=%d, want 1", src.closeCount())
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if src.closeCount() != 1 {
		t.Fatalf("second Close count=%d, want 1", src.closeCount())
	}
}

func TestWriteToCopiesUnderLock(t *testing.T) {
	data := []byte("cache-me")
	src := &closeRecorder{Reader: bytes.NewReader(data)}
	s := NewBufferedStream(src, int64(len(data)))
	defer s.Close()
	waitBuffered(t, s, len(data))
	var dest bytes.Buffer
	n, err := s.WriteTo(&dest)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(data)) || dest.String() != string(data) {
		t.Fatalf("n=%d dest=%q", n, dest.String())
	}
}
