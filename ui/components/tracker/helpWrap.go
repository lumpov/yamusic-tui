package tracker

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func wrapShortHelp(h help.Model, bindings []key.Binding) string {
	width := h.Width
	if width <= 0 {
		return h.ShortHelpView(bindings)
	}

	measure := h
	measure.Width = 0

	var lines []string
	var line []key.Binding
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		candidate := append(append([]key.Binding{}, line...), b)
		if len(line) > 0 && lipgloss.Width(measure.ShortHelpView(candidate)) > width {
			lines = append(lines, h.ShortHelpView(line))
			line = []key.Binding{b}
			continue
		}
		line = candidate
	}
	if len(line) > 0 {
		lines = append(lines, h.ShortHelpView(line))
	}
	return strings.Join(lines, "\n")
}
