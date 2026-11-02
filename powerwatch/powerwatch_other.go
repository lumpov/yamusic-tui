//go:build !darwin && !windows

package powerwatch

func Supported() bool { return false }

func ScreenLocked() bool { return false }

func Start(fn func(Event)) { setSend(fn) }

func Stop() { setSend(nil) }
