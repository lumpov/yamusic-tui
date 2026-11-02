package playlist

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/dece2183/yamusic-tui/config"
)

type helpKeyMap struct {
	CursorUp      key.Binding
	CursorDown    key.Binding
	Rename        key.Binding
	HidePlaylists key.Binding
	PlayerMode    key.Binding
	ShowHelp      key.Binding
	Renamable     bool
}

func newHelpMap() *helpKeyMap {
	controls := config.Current.Controls
	return &helpKeyMap{
		CursorUp:      key.NewBinding(controls.PlaylistsUp.Binding(), controls.PlaylistsUp.Help("up")),
		CursorDown:    key.NewBinding(controls.PlaylistsDown.Binding(), controls.PlaylistsDown.Help("down")),
		Rename:        key.NewBinding(controls.PlaylistsRename.Binding(), controls.PlaylistsRename.Help("rename")),
		HidePlaylists: key.NewBinding(controls.PlaylistsHide.Binding(), controls.PlaylistsHide.Help("hide")),
		PlayerMode:    key.NewBinding(controls.PlayerMode.Binding(), controls.PlayerMode.Help("mode")),
		ShowHelp:      key.NewBinding(controls.ShowAllKeys.Binding(), controls.ShowAllKeys.Help("show keys")),
	}
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.CursorUp, k.CursorDown, k.ShowHelp}
}

func (k helpKeyMap) FullHelp() [][]key.Binding {
	bindings := [][]key.Binding{
		{k.CursorUp, k.CursorDown},
	}

	if k.Renamable {
		bindings = append(bindings, []key.Binding{k.Rename})
	}

	bindings = append(bindings, []key.Binding{k.HidePlaylists, k.PlayerMode})

	return bindings
}
