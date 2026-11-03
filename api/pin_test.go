package api

import (
	"encoding/json"
	"testing"
)

func TestPinSeedsUnmarshal(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{`["artist:1","track:2"]`, []string{"artist:1", "track:2"}},
		{`"artist:1, track:2"`, []string{"artist:1", "track:2"}},
		{`"user:onyourwave"`, []string{"user:onyourwave"}},
		{`""`, nil},
		{`null`, nil},
		{`{}`, nil},
	}
	for _, tc := range cases {
		var s PinSeeds
		if err := json.Unmarshal([]byte(tc.raw), &s); err != nil {
			t.Fatalf("seeds %s: %v", tc.raw, err)
		}
		if len(s) != len(tc.want) {
			t.Fatalf("seeds %s: got %v, want %v", tc.raw, s, tc.want)
		}
		for i := range tc.want {
			if s[i] != tc.want[i] {
				t.Fatalf("seeds %s: got %v, want %v", tc.raw, s, tc.want)
			}
		}
	}
}

func TestPinHeaderUnmarshal(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`"Радио"`, "Радио"},
		{`{"title":"Unfamiliar"}`, "Unfamiliar"},
		{`{"name":"Collection"}`, "Collection"},
		{`{"text":"Chōchin"}`, "Chōchin"},
		{`null`, ""},
		{`[]`, ""},
	}
	for _, tc := range cases {
		var h PinHeader
		if err := json.Unmarshal([]byte(tc.raw), &h); err != nil {
			t.Fatalf("header %s: %v", tc.raw, err)
		}
		if string(h) != tc.want {
			t.Fatalf("header %s: got %q, want %q", tc.raw, h, tc.want)
		}
	}
}

func TestPinsDescWaveAndAlbum(t *testing.T) {
	raw := []byte(`{
		"pins": [
			{"type":"album_item","data":{"id":5,"title":"Hajime"}},
			{"type":"wave_item","data":{"title":"Chōchin","header":"My Vibe","seeds":["artist:123"],"stationId":"artist:123"}},
			{"type":"wave_item","data":{"header":{"title":"Unfamiliar · Calm"},"stationId":"mood:calm"}},
			{"type":"wave","data":{"seeds":"collection:1"}}
		]
	}`)
	var desc PinsDesc
	if err := json.Unmarshal(raw, &desc); err != nil {
		t.Fatal(err)
	}
	if desc.Pins[0].Data.Id != 5 || desc.Pins[0].Data.Title != "Hajime" {
		t.Fatalf("album pin: %+v", desc.Pins[0].Data)
	}

	var waves []PinInfo
	for _, pin := range desc.Pins {
		if isWavePin(pin) && len(pin.Data.WaveSeeds()) > 0 {
			waves = append(waves, pin)
		}
	}
	if len(waves) != 3 {
		t.Fatalf("waves=%d, want 3", len(waves))
	}
	if waves[0].Data.WaveTitle() != "Chōchin" {
		t.Fatalf("title=%q, want Chōchin", waves[0].Data.WaveTitle())
	}
	if got := waves[0].Data.WaveSeeds(); len(got) != 1 || got[0] != "artist:123" {
		t.Fatalf("seeds=%v, want [artist:123]", got)
	}
	if waves[1].Data.WaveTitle() != "Unfamiliar · Calm" {
		t.Fatalf("title=%q, want Unfamiliar · Calm", waves[1].Data.WaveTitle())
	}
	if got := waves[1].Data.WaveSeeds(); len(got) != 1 || got[0] != "mood:calm" {
		t.Fatalf("seeds=%v, want [mood:calm]", got)
	}
	if waves[2].Data.WaveTitle() != "collection:1" {
		t.Fatalf("title=%q, want collection:1", waves[2].Data.WaveTitle())
	}
}

func TestRotorNewSessionSeedsEmpty(t *testing.T) {
	c := &YaMusicClient{}
	if _, err := c.RotorNewSessionSeeds(nil); err == nil {
		t.Fatal("empty seeds must fail")
	}
	if _, err := c.RotorNewSessionSeeds([]string{}); err == nil {
		t.Fatal("empty seeds must fail")
	}
}
