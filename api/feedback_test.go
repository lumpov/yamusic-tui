package api

import "testing"

func TestTrackRefWithoutAlbum(t *testing.T) {
	if got := trackRef(&Track{Id: "t1"}); got != "t1" {
		t.Fatalf("trackRef without album = %q, want t1", got)
	}
}

func TestTrackRefWithAlbum(t *testing.T) {
	if got := trackRef(&Track{Id: "t1", Albums: []Album{{Id: 9}}}); got != "t1:9" {
		t.Fatalf("trackRef with album = %q, want t1:9", got)
	}
}

func TestNewTrackFeedbackEventWithoutAlbum(t *testing.T) {
	ev := NewTrackFeedbackEvent(EV_TRACK_STARTED, &Track{Id: "t1", DurationMs: 180000}, 0)
	if ev.TrackId != "t1" {
		t.Fatalf("TrackId=%q, want t1", ev.TrackId)
	}
	if ev.TrackLengthSeconds != 180 {
		t.Fatalf("TrackLengthSeconds=%v, want 180", ev.TrackLengthSeconds)
	}
}

func TestNewTrackFeedbackEventNilTrack(t *testing.T) {
	ev := NewTrackFeedbackEvent(EV_TRACK_STARTED, nil, 1.5)
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.TrackId != "" {
		t.Fatalf("TrackId=%q, want empty", ev.TrackId)
	}
	if ev.TrackLengthSeconds != 0 {
		t.Fatalf("TrackLengthSeconds=%v, want 0", ev.TrackLengthSeconds)
	}
	if ev.TotalPlayedSeconds != 1.5 {
		t.Fatalf("TotalPlayedSeconds=%v, want 1.5", ev.TotalPlayedSeconds)
	}
}
