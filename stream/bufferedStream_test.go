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

type blockingCloser struct {
	readStarted chan struct{}
	unblock     chan struct{}
	mu          sync.Mutex
	closed      int
}

func (b *blockingCloser) Read(p []byte) (int, error) {
	select {
	case <-b.readStarted:
	default:
		close(b.readStarted)
	}
	<-b.unblock
	return 0, io.EOF
}

func (b *blockingCloser) Close() error {
	b.mu.Lock()
	b.closed++
	b.mu.Unlock()
	select {
	case <-b.unblock:
	default:
		close(b.unblock)
	}
	return nil
}

func TestCloseDoesNotWaitForSourceRead(t *testing.T) {
	src := &blockingCloser{
		readStarted: make(chan struct{}),
		unblock:     make(chan struct{}),
	}
	s := NewBufferedStream(src, 1<<20)
	select {
	case <-src.readStarted:
	case <-time.After(time.Second):
		t.Fatal("source Read did not start")
	}
	done := make(chan error, 1)
	go func() {
		done <- s.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Close waited for source Read")
	}
}

func TestBufferCompleteClearsErrorAndClosesSource(t *testing.T) {
	data := bytes.Repeat([]byte("ab"), 50)
	src := &closeRecorder{Reader: bytes.NewReader(data)}
	s := NewBufferedStream(src, int64(len(data)))
	defer s.Close()
	waitBuffered(t, s, len(data))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if src.closeCount() >= 1 && s.Error() == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("closeCount=%d error=%v", src.closeCount(), s.Error())
}

func TestShortBodySetsBufferError(t *testing.T) {
	src := &closeRecorder{Reader: bytes.NewReader([]byte("xy"))}
	s := NewBufferedStream(src, 64)
	defer s.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if s.Error() == io.ErrUnexpectedEOF {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Error=%v, want unexpected EOF", s.Error())
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
