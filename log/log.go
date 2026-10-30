package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dece2183/yamusic-tui/config"
)

type Level int

const (
	LVL_PANIC Level = iota
	LVL_ERROR
	LVL_WARNIGN
	LVL_INFO
)

var lvlName = map[Level]string{
	LVL_PANIC:   "PANIC",
	LVL_ERROR:   "ERROR",
	LVL_WARNIGN: "WARN",
	LVL_INFO:    "INFO",
}

var (
	file *os.File
)

const (
	_FILE_PERM os.FileMode = 0600
	_DIR_PERM  os.FileMode = 0700
)

func getLogLocation() (string, error) {
	tempDir := filepath.Join(os.TempDir(), config.DirName)
	if err := os.MkdirAll(tempDir, _DIR_PERM); err != nil {
		return "", err
	}
	if err := os.Chmod(tempDir, _DIR_PERM); err != nil {
		return "", err
	}
	return filepath.Join(tempDir, config.DirName+".log"), nil
}

func Location() string {
	if file == nil {
		return ""
	}

	return file.Name()
}

func Start() {
	path, err := getLogLocation()
	if err != nil {
		return
	}

	Stop()
	_ = os.Remove(path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, _FILE_PERM)
	if err != nil {
		return
	}
	if err := f.Chmod(_FILE_PERM); err != nil {
		f.Close()
		return
	}
	file = f
}

func Stop() {
	if file == nil {
		return
	}
	file.Close()
	file = nil
}

func Print(lvl Level, format string, args ...any) {
	if file == nil {
		return
	}

	t := time.Now()
	format = fmt.Sprintf("[ %s ][(%d) %02d:%02d:%02d] - ", lvlName[lvl], t.Day(), t.Hour(), t.Minute(), t.Second()) + format + "\n"
	fmt.Fprintf(file, format, args...)
	file.Sync()
}
