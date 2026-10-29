package tracker

import "testing"

func TestCompressedByteOffsetZeroDuration(t *testing.T) {
	if got := compressedByteOffset(1000, 0, 10); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestCompressedByteOffsetMidTrack(t *testing.T) {
	got := compressedByteOffset(1000, 1000, 500)
	if got != 500 {
		t.Fatalf("got %d, want 500", got)
	}
}

func TestCompressedByteOffsetNegativeRewind(t *testing.T) {
	got := compressedByteOffset(1000, 1000, -100)
	if got >= 0 {
		t.Fatalf("negative rewind must stay negative, got %d", got)
	}
}

func TestClampByteOffset(t *testing.T) {
	if got := clampByteOffset(-10, 100); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
	if got := clampByteOffset(150, 100); got != 100 {
		t.Fatalf("got %d, want 100", got)
	}
	if got := clampByteOffset(40, 100); got != 40 {
		t.Fatalf("got %d, want 40", got)
	}
}

func TestClampPlayableByteOffsetKeepsRewindStep(t *testing.T) {
	if got := clampPlayableByteOffset(-10, 1000, 50); got != 0 {
		t.Fatalf("start: got %d, want 0", got)
	}
	if got := clampPlayableByteOffset(1000, 1000, 50); got != 950 {
		t.Fatalf("end: got %d, want 950", got)
	}
	if got := clampPlayableByteOffset(2000, 100, 200); got != 0 {
		t.Fatalf("step larger than track: got %d, want 0", got)
	}
}
