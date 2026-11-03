//go:build linux

package idleprevent

import (
	"os"

	"github.com/dece2183/yamusic-tui/log"
	"github.com/godbus/dbus/v5"
)

var inhibitFile *os.File

func Supported() bool { return true }

func acquire() bool {
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Print(log.LVL_WARNIGN, "idleprevent logind bus: %s", err)
		return false
	}
	obj := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	var fd dbus.UnixFD
	err = obj.Call("org.freedesktop.login1.Manager.Inhibit", 0, "idle", "yamusic-tui", "always-on playback", "block").Store(&fd)
	if err != nil {
		log.Print(log.LVL_WARNIGN, "idleprevent logind: %s", err)
		return false
	}
	inhibitFile = os.NewFile(uintptr(fd), "logind-inhibit")
	return true
}

func release() {
	if inhibitFile == nil {
		return
	}
	_ = inhibitFile.Close()
	inhibitFile = nil
}
