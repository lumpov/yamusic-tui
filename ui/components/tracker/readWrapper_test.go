package tracker

import (
	"bytes"
	"io"
	"testing"

	"github.com/dece2183/yamusic-tui/stream"
)

func TestReadWrapperProgressAndEOF(t *testing.T) {
	data := bytes.Repeat([]byte{1}, 64)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{program: nil, emitNextOnEOF: true}
	w.NewReader(buf)

	dest := make([]byte, 64)
	for i := 0; i < 8; i++ {
		_, err := w.Read(dest)
		if err == io.EOF {
			break
		}
	}

	if w.Progress() < 0 || w.Progress() > 1 {
		t.Fatalf("progress out of range: %f", w.Progress())
	}
	if w.Length() != int64(len(data)) {
		t.Fatalf("length=%d", w.Length())
	}
}

func TestReadWrapperNilProgramDoesNotPanic(t *testing.T) {
	w := &readWrapper{}
	w.sendNext()
	w.send(STOP)
}

func TestReadWrapperSeekClearsTrackDone(t *testing.T) {
	data := bytes.Repeat([]byte{1}, 32)
	buf := stream.NewBufferedStream(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
	defer buf.Close()

	w := &readWrapper{program: nil, source: buf, trackBuffer: buf, emitNextOnEOF: false}
	dest := make([]byte, 8)
	for i := 0; i < 16; i++ {
		_, err := w.Read(dest)
		if err == io.EOF {
			break
		}
	}
	if !w.trackDone {
		t.Fatal("full read must set trackDone")
	}
	if _, err := w.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if w.trackDone {
		t.Fatal("Seek must clear trackDone")
	}
}
