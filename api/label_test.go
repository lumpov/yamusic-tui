package api

import (
	"encoding/json"
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
