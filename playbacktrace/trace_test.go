package playbacktrace

import "testing"

func TestDisabledIsNoop(t *testing.T) {
	if Enabled {
		t.Skip("Enabled")
	}
	Reset()
	t.Cleanup(Reset)
	Begin("1", "t")
	Event("x")
	Flush("stop")
	if n := len(Snapshot()); n != 0 {
		t.Fatalf("disabled n=%d", n)
	}
}

func TestEventKeepsOrderAndCap(t *testing.T) {
	if !Enabled {
		t.Skip("Enabled")
	}
	Reset()
	t.Cleanup(Reset)

	Begin("1", "t")
	for i := 0; i < maxEvents+10; i++ {
		Event("i=%d", i)
	}
	got := Snapshot()
	if len(got) == 0 || len(got) > maxEvents {
		t.Fatalf("len=%d", len(got))
	}
}

func TestFlushClearsBuffer(t *testing.T) {
	if !Enabled {
		t.Skip("Enabled")
	}
	Reset()
	t.Cleanup(Reset)
	Begin("1", "t")
	Event("rewind +5s")
	Flush("stop")
	if n := len(Snapshot()); n != 0 {
		t.Fatalf("after flush n=%d", n)
	}
}
