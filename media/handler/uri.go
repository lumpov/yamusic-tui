package handler

import (
	"net/url"
	"strings"
)

func YandexTrackIdFromURI(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", false
	}
	if u.Scheme != "https" {
		return "", false
	}
	if strings.ToLower(u.Hostname()) != "music.yandex.ru" {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 4 && parts[0] == "album" && parts[2] == "track" && parts[3] != "" {
		return parts[3], true
	}
	return "", false
}
