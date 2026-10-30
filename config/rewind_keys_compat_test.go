package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var bubbleteaArrowSeq = map[string]string{
	"\x1b[C":    "right",
	"\x1b[D":    "left",
	"\x1b[A":    "up",
	"\x1b[B":    "down",
	"\x1b[1;5C": "ctrl+right",
	"\x1b[1;5D": "ctrl+left",
	"\x1b[1;5A": "ctrl+up",
	"\x1b[1;5B": "ctrl+down",
	"\x1b[1;2C": "shift+right",
	"\x1b[1;2D": "shift+left",
	"\x1b[1;3C": "alt+right",
	"\x1b[1;3D": "alt+left",
	"\x1bOC":    "shift+right",
	"\x1b[Oc":   "alt+ctrl+right",
	"\x1b[1;7C": "alt+ctrl+right",
	"\x1bf":     "alt+f",
	"\x1bb":     "alt+b",
}

func TestRewindKeysAreShiftArrows(t *testing.T) {
	InitTest(t)

	shiftRight := tea.KeyMsg{Type: tea.KeyShiftRight}.String()
	if shiftRight != "shift+right" {
		t.Fatalf("KeyShiftRight.String()=%q, want shift+right", shiftRight)
	}
	if !Current.Controls.PlayerRewindForward.Contains(shiftRight) {
		t.Fatalf("player-rewind-forward не совпадает с %q", shiftRight)
	}
	if Current.Controls.PlayerRewindForward.Contains("right") {
		t.Fatal("player-rewind-forward не должен ловить right")
	}
	if Current.Controls.PlayerRewindForward.Contains("ctrl+right") {
		t.Fatal("player-rewind-forward не должен ловить ctrl+right")
	}
	if Current.Controls.PlayerRewindForward.Contains("alt+right") {
		t.Fatal("player-rewind-forward не должен ловить alt+right")
	}

	shiftLeft := tea.KeyMsg{Type: tea.KeyShiftLeft}.String()
	if shiftLeft != "shift+left" {
		t.Fatalf("KeyShiftLeft.String()=%q, want shift+left", shiftLeft)
	}
	if !Current.Controls.PlayerRewindBackward.Contains(shiftLeft) {
		t.Fatalf("player-rewind-backward не совпадает с %q", shiftLeft)
	}

	plainRight := tea.KeyMsg{Type: tea.KeyRight}.String()
	if !Current.Controls.PlayerNext.Contains(plainRight) {
		t.Fatalf("player-next должен ловить %q", plainRight)
	}
	if Current.Controls.PlayerNext.Contains("shift+right") {
		t.Fatal("player-next не должен ловить shift+right")
	}
}

func TestRewindTerminalSequences(t *testing.T) {
	InitTest(t)

	cases := []struct {
		seq    string
		name   string
		rewind bool
		next   bool
	}{
		{seq: "\x1b[1;2C", name: "shift+right", rewind: true, next: false},
		{seq: "\x1b[C", name: "right", rewind: false, next: true},
		{seq: "\x1b[1;5C", name: "ctrl+right", rewind: false, next: false},
		{seq: "\x1b[1;3C", name: "alt+right", rewind: false, next: false},
		{seq: "\x1bf", name: "alt+f", rewind: false, next: false},
	}
	for _, tc := range cases {
		got := bubbleteaArrowSeq[tc.seq]
		if got != tc.name {
			t.Fatalf("seq %q → %q, want %q", tc.seq, got, tc.name)
		}
		rewind := Current.Controls.PlayerRewindForward.Contains(got)
		next := Current.Controls.PlayerNext.Contains(got)
		if rewind != tc.rewind || next != tc.next {
			t.Fatalf("%q: rewind=%v next=%v, want rewind=%v next=%v",
				got, rewind, next, tc.rewind, tc.next)
		}
	}
}

func TestDarwinShiftRightNotTakenByMissionControl(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("не macOS")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("хост CI, не рабочий Mac")
	}

	enabled, found, err := darwinHotkeyEnabled("81")
	if err != nil {
		t.Fatal(err)
	}
	if found && enabled {
		t.Log("hotkey 81 (Ctrl+→ / рабочий стол) включён — на shift+right не влияет")
	}
}

func TestProbeShiftRightTerminal(t *testing.T) {
	if os.Getenv("YAMUSIC_PROBE_KEYS") != "1" {
		t.Skip("запуск: YAMUSIC_PROBE_KEYS=1 go test ./config -run TestProbeShiftRightTerminal -v")
	}
	if runtime.GOOS == "windows" {
		t.Skip("интерактивный опрос байтов — unix TTY")
	}
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		t.Skip("stdin не терминал")
	}

	InitTest(t)
	restore, err := unixRawStdin()
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	t.Log("нажми Shift+→, 8 секунд")
	buf := make([]byte, 32)
	nch := make(chan int, 1)
	go func() {
		n, _ := os.Stdin.Read(buf)
		nch <- n
	}()

	var n int
	select {
	case n = <-nch:
	case <-time.After(8 * time.Second):
		t.Fatal("за 8с байты не пришли: Shift+→ не доходит до процесса")
	}
	if n == 0 {
		t.Fatal("прочитано 0 байт")
	}

	seq := string(buf[:n])
	name, ok := bubbleteaArrowSeq[seq]
	if !ok {
		t.Fatalf("байты %q не из таблицы Bubble Tea v1.3.4, rewind не сработает", seq)
	}
	if !Current.Controls.PlayerRewindForward.Contains(name) {
		t.Fatalf("терминал дал %q (%q), конфиг ждёт shift+right; player-next=%v",
			name, seq, Current.Controls.PlayerNext.Contains(name))
	}
}

type darwinHotkeyFile struct {
	AppleSymbolicHotKeys map[string]darwinHotkey `json:"AppleSymbolicHotKeys"`
}

type darwinHotkey struct {
	Enabled json.RawMessage `json:"enabled"`
}

func darwinHotkeyEnabled(id string) (enabled bool, found bool, err error) {
	path := filepath.Join(os.Getenv("HOME"), "Library/Preferences/com.apple.symbolichotkeys.plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", path).Output()
	if err != nil {
		return false, false, err
	}
	var file darwinHotkeyFile
	if err := json.Unmarshal(out, &file); err != nil {
		return false, false, err
	}
	entry, ok := file.AppleSymbolicHotKeys[id]
	if !ok {
		return false, false, nil
	}
	var b bool
	if json.Unmarshal(entry.Enabled, &b) == nil {
		return b, true, nil
	}
	var n float64
	if json.Unmarshal(entry.Enabled, &n) == nil {
		return n != 0, true, nil
	}
	return false, true, nil
}

func unixRawStdin() (func(), error) {
	sttyG := exec.Command("stty", "-g")
	sttyG.Stdin = os.Stdin
	old, err := sttyG.Output()
	if err != nil {
		return nil, err
	}
	raw := exec.Command("stty", "raw", "-echo", "min", "1", "time", "0")
	raw.Stdin = os.Stdin
	if err := raw.Run(); err != nil {
		return nil, err
	}
	return func() {
		cmd := exec.Command("stty", string(trimNL(old)))
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}, nil
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
