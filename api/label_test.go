package api

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestLabelIDAcceptsNumber(t *testing.T) {
	var label Label
	if err := json.Unmarshal([]byte(`{"id":37458779,"name":"N"}`), &label); err != nil {
		t.Fatal(err)
	}
	if label.Id != "37458779" {
		t.Fatalf("Id=%q, want 37458779", label.Id)
	}
	if label.Name != "N" {
		t.Fatalf("Name=%q, want N", label.Name)
	}
}

func TestLabelIDAcceptsString(t *testing.T) {
	var label Label
	if err := json.Unmarshal([]byte(`{"id":"abc","name":"N"}`), &label); err != nil {
		t.Fatal(err)
	}
	if label.Id != "abc" {
		t.Fatalf("Id=%q, want abc", label.Id)
	}
}

func TestTrackLyricsLyricIdAcceptsNumber(t *testing.T) {
	var lyrics TrackLyrics
	err := json.Unmarshal([]byte(`{"downloadUrl":"https://x","lyricId":8145339,"externalLyricId":"8638863"}`), &lyrics)
	if err != nil {
		t.Fatal(err)
	}
	if lyrics.LyricId != "8145339" {
		t.Fatalf("LyricId=%q, want 8145339", lyrics.LyricId)
	}
	if lyrics.ExternalLyricId != "8638863" {
		t.Fatalf("ExternalLyricId=%q, want 8638863", lyrics.ExternalLyricId)
	}
}

func TestHttpStatusErrorCode(t *testing.T) {
	err := &httpStatusError{Code: 404, Status: "404 Not Found"}
	if statusCode(err) != 404 {
		t.Fatalf("statusCode=%d, want 404", statusCode(err))
	}
	if statusCode(errors.New("nope")) != 0 {
		t.Fatal("non-status error must be 0")
	}
}

func TestAlbumLabelsNumberID(t *testing.T) {
	var album Album
	err := json.Unmarshal([]byte(`{"id":1,"title":"A","labels":[{"id":4722,"name":"L"}]}`), &album)
	if err != nil {
		t.Fatal(err)
	}
	if len(album.Labels) != 1 || album.Labels[0].Id != "4722" {
		t.Fatalf("labels=%v", album.Labels)
	}
}
