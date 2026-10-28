package api

import "testing"

func TestParseLRCTextSynced(t *testing.T) {
	got := parseLRCText("[00:01.00]one\n[00:02.50]two\n")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Timestamp != 1000 || got[0].Line != "one" {
		t.Fatalf("first=%v", got[0])
	}
	if got[1].Timestamp != 2050 || got[1].Line != "two" {
		t.Fatalf("second=%v", got[1])
	}
}

func TestParseLRCTextPlainFallback(t *testing.T) {
	got := parseLRCText("line a\n\nline b\n")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Timestamp != 0 || got[0].Line != "line a" {
		t.Fatalf("first=%v", got[0])
	}
	if got[1].Line != "line b" {
		t.Fatalf("second=%v", got[1])
	}
}
