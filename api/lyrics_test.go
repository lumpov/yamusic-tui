package api

import (
	"encoding/json"
	"testing"
)

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

func TestUnmarshalTrackSupplementFullLyrics(t *testing.T) {
	var sup TrackSupplement
	err := json.Unmarshal([]byte(`{"lyrics":{"id":1,"lyrics":"short","fullLyrics":"line a\nline b","hasRights":true,"showTranslation":false,"textLanguage":"en"}}`), &sup)
	if err != nil {
		t.Fatal(err)
	}
	text := supplementLyricsText(sup)
	if text != "line a\nline b" {
		t.Fatalf("text=%q", text)
	}
	got := parseLRCText(text)
	if len(got) != 2 || got[0].Line != "line a" || got[1].Line != "line b" {
		t.Fatalf("parsed=%v", got)
	}
}

func TestSupplementLyricsTextNil(t *testing.T) {
	if got := supplementLyricsText(TrackSupplement{}); got != "" {
		t.Fatalf("got=%q", got)
	}
}
