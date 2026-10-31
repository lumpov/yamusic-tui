package handler

import "testing"

func TestYandexTrackIdFromURI(t *testing.T) {
	id, ok := YandexTrackIdFromURI("https://music.yandex.ru/album/123/track/456")
	if !ok || id != "456" {
		t.Fatalf("share link: id=%q ok=%v", id, ok)
	}
	if _, ok := YandexTrackIdFromURI("https://music.yandex.ru/track/456"); ok {
		t.Fatal("bare /track must be rejected")
	}
	if _, ok := YandexTrackIdFromURI("file:///tmp/a.mp3"); ok {
		t.Fatal("file uri must be rejected")
	}
	if _, ok := YandexTrackIdFromURI("https://example.com/album/1/track/2"); ok {
		t.Fatal("foreign host must be rejected")
	}
	if _, ok := YandexTrackIdFromURI("http://music.yandex.ru/album/1/track/2"); ok {
		t.Fatal("http must be rejected")
	}
	if _, ok := YandexTrackIdFromURI(""); ok {
		t.Fatal("empty uri must be rejected")
	}
}
