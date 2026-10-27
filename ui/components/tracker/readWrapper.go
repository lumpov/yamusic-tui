package tracker

import (
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dece2183/yamusic-tui/log"
	"github.com/dece2183/yamusic-tui/stream"
)

const (
	_PROGRESS_UPDATE_PERIOD = 850 * time.Millisecond
)

type readWrapper struct {
	program        *tea.Program
	source         io.ReadSeeker
	trackBuffer    *stream.BufferedStream
	trackBuffered  bool
	trackDone      bool
	emitNextOnEOF  bool
	lastUpdateTime time.Time
}

func (w *readWrapper) NewReader(reader *stream.BufferedStream) {
	w.trackBuffered = false
	w.trackDone = false
	w.emitNextOnEOF = true
	w.trackBuffer = reader
	w.source = newPlaybackSource(reader)
	w.lastUpdateTime = time.Now()
}

func (w *readWrapper) Close() {
	if w.trackBuffer != nil {
		w.trackBuffer.Close()
	}
}

func (w *readWrapper) Read(dest []byte) (n int, err error) {
	if w.trackBuffer == nil || w.source == nil {
		err = io.EOF
		return
	}

	n, err = w.source.Read(dest)
	if err != nil && err != io.EOF {
		if w.trackBuffer.Error() != nil {
			err = w.trackBuffer.Error()
			log.Print(log.LVL_ERROR, "buffering error: %s", err)
			w.send(STOP)
			return
		}
		log.Print(log.LVL_WARNIGN, "mp3 decoding error: %s", err)
		err = nil
	}

	w.afterRead()
	return
}

func (w *readWrapper) Seek(offset int64, whence int) (int64, error) {
	w.lastUpdateTime = time.Now()
	if w.source == nil {
		return 0, io.EOF
	}
	return w.source.Seek(offset, whence)
}

func (w *readWrapper) Length() int64 {
	return w.trackBuffer.Length()
}

func (w *readWrapper) Progress() float64 {
	return w.trackBuffer.Progress()
}

func (w *readWrapper) afterRead() {
	if w.trackBuffer.IsBuffered() && !w.trackBuffered {
		w.trackBuffered = true
		w.send(BUFFERING_COMPLETE)
	}

	if w.trackBuffer.IsDone() && !w.trackDone {
		w.trackDone = true
		if w.emitNextOnEOF {
			w.sendNext()
		}
		return
	}

	if !w.trackDone && time.Since(w.lastUpdateTime) > _PROGRESS_UPDATE_PERIOD {
		w.lastUpdateTime = time.Now()
		w.send(ProgressControl(w.trackBuffer.Progress()))
	}
}

func (w *readWrapper) sendNext() {
	w.send(NEXT)
}

func (w *readWrapper) send(msg tea.Msg) {
	if w.program == nil {
		return
	}
	go w.program.Send(msg)
}
