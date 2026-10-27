package tracker

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/dece2183/yamusic-tui/config"
)

func NewForTest() *Model {
	config.InitTest()
	likes := map[string]bool{}
	fp := &fakePlayer{}
	m := &Model{
		likesMap:     &likes,
		progress:     progress.New(),
		volumeBar:    progress.New(),
		help:         help.New(),
		helpMap:      newHelpMap(),
		paused:       true,
		trackWrapper: &readWrapper{},
		audio:        &fakeBackend{player: fp},
	}
	go m.volumeFader()
	return m
}
