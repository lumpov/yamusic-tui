package stream

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dece2183/yamusic-tui/playbacktrace"
)

const (
	_BUFFERING_AMOUNT = 32 * 1024
	_BUFFERING_PERIOD = 100 * time.Millisecond
)

var errOutOfSize = errors.New("position is out of data size")

type BufferedStream struct {
	source       io.ReadCloser
	bufferTimer  *time.Ticker
	closed       chan bool
	lastError    error
	readBuffer   []byte
	readIndex    int64
	totalSize    int64
	buffered     bool
	done         bool
	sourceClosed bool
	mux          sync.Mutex
	sourceMu     sync.Mutex
}

func NewBufferedStream(source io.ReadCloser, totalSize int64) *BufferedStream {
	rs := BufferedStream{
		source:      source,
		totalSize:   totalSize,
		bufferTimer: time.NewTicker(_BUFFERING_PERIOD),
		closed:      make(chan bool),
	}

	go rs.bufferFrames(_BUFFERING_AMOUNT)
	return &rs
}

func (h *BufferedStream) Length() int64 {
	return int64(h.totalSize)
}

func (h *BufferedStream) Close() error {
	var err error

	h.mux.Lock()
	defer h.mux.Unlock()

	h.readBuffer = nil
	h.stopBuffering()
	h.done = true
	err = h.closeSource()

	return err
}

func (h *BufferedStream) closeSource() error {
	if h.sourceClosed || h.source == nil {
		return nil
	}
	h.sourceClosed = true
	return h.source.Close()
}

func (h *BufferedStream) finishReadLocked(err error) error {
	if err == io.EOF || err == http.ErrBodyReadAfterClose {
		if !h.sourceClosed {
			h.closeSource()
		}
		h.stopBuffering()
		h.done = true
		if h.lastError == io.EOF {
			h.lastError = nil
		}
		return io.EOF
	}
	if err != nil {
		h.lastError = err
		return err
	}
	h.lastError = nil
	return nil
}

func (h *BufferedStream) readSource(p []byte) (int, error) {
	h.sourceMu.Lock()
	defer h.sourceMu.Unlock()
	if h.source == nil {
		return 0, io.EOF
	}
	t0 := time.Now()
	n, err := h.source.Read(p)
	if d := time.Since(t0); d > 400*time.Millisecond {
		playbacktrace.Event("source read wait_ms=%d n=%d err=%v", d.Milliseconds(), n, err)
	}
	return n, err
}

func (h *BufferedStream) readSourceFull(p []byte) (int, error) {
	h.sourceMu.Lock()
	defer h.sourceMu.Unlock()
	if h.source == nil {
		return 0, io.EOF
	}
	t0 := time.Now()
	n, err := io.ReadFull(h.source, p)
	if d := time.Since(t0); d > 400*time.Millisecond {
		playbacktrace.Event("source readFull wait_ms=%d n=%d need=%d err=%v", d.Milliseconds(), n, len(p), err)
	}
	return n, err
}

func (h *BufferedStream) Read(dest []byte) (n int, err error) {
	destLen := int64(len(dest))

	h.mux.Lock()
	readBufLen := int64(len(h.readBuffer))

	if h.readIndex < readBufLen {
		endIndex := h.readIndex + destLen
		if endIndex > readBufLen {
			endIndex = readBufLen
		}
		bufferedPart := h.readBuffer[h.readIndex:endIndex]
		if destLen-int64(len(bufferedPart)) <= 0 {
			copy(dest, bufferedPart)
			n = len(bufferedPart)
			h.readIndex += int64(n)
			if h.readIndex >= h.totalSize {
				err = io.EOF
			}
			err = h.finishReadLocked(err)
			h.mux.Unlock()
			return
		}

		copy(dest, bufferedPart)
		n = len(bufferedPart)
		h.readIndex += int64(n)
		h.mux.Unlock()

		var unbufferedLen int
		unbufferedLen, err = h.readSource(dest[n:])

		h.mux.Lock()
		if !h.done && !h.sourceClosed && unbufferedLen > 0 {
			h.readBuffer = append(h.readBuffer, dest[n:n+unbufferedLen]...)
			n += unbufferedLen
			h.readIndex += int64(unbufferedLen)
		}
		if h.readIndex >= h.totalSize {
			err = io.EOF
		}
		err = h.finishReadLocked(err)
		h.mux.Unlock()
		return
	}

	gap := h.readIndex - readBufLen
	need := gap + destLen
	h.mux.Unlock()

	newFrame := make([]byte, need)
	n, err = h.readSourceFull(newFrame)

	h.mux.Lock()
	if h.done || h.sourceClosed {
		h.mux.Unlock()
		return n, err
	}
	if n > 0 {
		h.readBuffer = append(h.readBuffer, newFrame[:n]...)
	}
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	if h.readIndex < int64(len(h.readBuffer)) {
		copy(dest, h.readBuffer[h.readIndex:])
	} else {
		err = io.EOF
	}
	h.readIndex += int64(n) - gap
	err = h.finishReadLocked(err)
	h.mux.Unlock()
	return
}

func (h *BufferedStream) Seek(offset int64, whence int) (pos int64, err error) {
	h.mux.Lock()

	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos = h.readIndex + offset
	case io.SeekEnd:
		pos = h.totalSize + offset
	}

	if pos < 0 || pos > h.totalSize {
		pos = h.readIndex
		err = errOutOfSize
	} else {
		if pos == h.totalSize {
			h.done = true
		} else {
			h.done = false
		}
		h.readIndex = pos
	}

	h.mux.Unlock()
	return
}

func (h *BufferedStream) IsDone() bool {
	if h == nil {
		return false
	}
	return h.done
}

func (h *BufferedStream) IsBuffered() bool {
	if h == nil {
		return false
	}
	return h.buffered
}

func (h *BufferedStream) Progress() float64 {
	if h == nil {
		return 0
	}
	return float64(h.readIndex) / float64(h.totalSize)
}

func (h *BufferedStream) BufferingProgress() float64 {
	if h == nil {
		return 0
	}
	return float64(len(h.readBuffer)) / float64(h.totalSize)
}

func (h *BufferedStream) BufferAll() {
	h.mux.Lock()
	if h.buffered {
		h.mux.Unlock()
		return
	}
	h.stopBuffering()
	h.mux.Unlock()

	h.sourceMu.Lock()
	var newFrame []byte
	var err error
	if h.source != nil {
		newFrame, err = io.ReadAll(h.source)
	}
	h.sourceMu.Unlock()

	h.mux.Lock()
	defer h.mux.Unlock()
	if h.done {
		return
	}
	if err != nil {
		h.lastError = err
		return
	}
	h.readBuffer = append(h.readBuffer, newFrame...)
	h.closeSource()
}

func (h *BufferedStream) WriteTo(dest io.Writer) (int64, error) {
	h.mux.Lock()
	defer h.mux.Unlock()
	n, err := dest.Write(h.readBuffer)
	return int64(n), err
}

func (h *BufferedStream) Error() error {
	h.mux.Lock()
	defer h.mux.Unlock()
	if h.lastError == io.EOF || h.lastError == http.ErrBodyReadAfterClose {
		return nil
	}
	return h.lastError
}

func (h *BufferedStream) stopBuffering() {
	h.buffered = true
	if h.closed != nil {
		h.bufferTimer.Stop()
		close(h.closed)
		h.closed = nil
	}
}

func (h *BufferedStream) bufferFrames(size int64) {
	buf := make([]byte, size)
	for {
		h.mux.Lock()
		if h.buffered || h.done || h.totalSize <= int64(len(h.readBuffer)) {
			nbuf := len(h.readBuffer)
			h.stopBuffering()
			_ = h.closeSource()
			h.mux.Unlock()
			playbacktrace.Event("buffer done n=%d size=%d", nbuf, h.totalSize)
			return
		}
		h.mux.Unlock()

		n, err := h.readSourceFull(buf)

		h.mux.Lock()
		if h.done || h.buffered {
			h.mux.Unlock()
			return
		}
		if n > 0 {
			h.readBuffer = append(h.readBuffer, buf[:n]...)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			short := h.totalSize > 0 && int64(len(h.readBuffer)) < h.totalSize
			nbuf := len(h.readBuffer)
			h.stopBuffering()
			_ = h.closeSource()
			if short {
				h.lastError = io.ErrUnexpectedEOF
			} else if h.lastError == io.EOF {
				h.lastError = nil
			}
			h.mux.Unlock()
			playbacktrace.Event("buffer eof n=%d size=%d short=%v err=%v", nbuf, h.totalSize, short, h.Error())
			return
		}
		if err != nil && h.lastError == nil {
			playbacktrace.Event("buffer read err=%v n=%d size=%d", err, len(h.readBuffer), h.totalSize)
		}
		h.lastError = err
		h.mux.Unlock()

		select {
		case <-h.bufferTimer.C:
			continue
		case <-h.closed:
			return
		}
	}
}
