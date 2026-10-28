package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDurationMsToSeconds(t *testing.T) {
	cases := []struct {
		ms   int
		want int
	}{
		{0, 0},
		{-1, 0},
		{999, 0},
		{1000, 1},
		{180000, 180},
		{181000, 181},
	}
	for _, tc := range cases {
		if got := durationMsToSeconds(tc.ms); got != tc.want {
			t.Fatalf("durationMsToSeconds(%d)=%d, want %d", tc.ms, got, tc.want)
		}
	}
}

func TestProccessRequestReturnsDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "not json")
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = proccessRequest[map[string]string](req)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
	if _, ok := err.(*json.SyntaxError); !ok && !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("unexpected error %T %v", err, err)
	}
}

func TestProccessRequestDecodesOKResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"invocationInfo":{"req-id":"r1"},"result":{"id":"t1"}}`)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, inv, err := proccessRequest[Track](req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Id != "t1" {
		t.Fatalf("result.Id=%q, want t1", result.Id)
	}
	if inv.ReqId != "r1" {
		t.Fatalf("inv.ReqId=%q, want r1", inv.ReqId)
	}
}

func TestProccessRequestNotFoundStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "missing")
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = proccessRequest[Track](req)
	if statusCode(err) != http.StatusNotFound {
		t.Fatalf("statusCode=%d err=%v, want 404", statusCode(err), err)
	}
}

func TestPlayTrackNilTrack(t *testing.T) {
	client := &YaMusicClient{}
	if err := client.PlayTrack(nil, false); err == nil {
		t.Fatal("expected error for nil track")
	}
}
