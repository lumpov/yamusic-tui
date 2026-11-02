//go:build darwin

package powerwatch

/*
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework CoreGraphics
void startPowerWatch(void);
void stopPowerWatch(void);
int queryScreenLocked(void);
*/
import "C"

//export goPowerDisplayOff
func goPowerDisplayOff() { emit(DisplayOff) }

//export goPowerDisplayOn
func goPowerDisplayOn() { emit(DisplayOn) }

//export goPowerSessionLocked
func goPowerSessionLocked() { emit(SessionLocked) }

//export goPowerSessionUnlocked
func goPowerSessionUnlocked() { emit(SessionUnlocked) }

func Supported() bool { return true }

func ScreenLocked() bool { return C.queryScreenLocked() != 0 }

func Start(fn func(Event)) {
	setSend(fn)
	C.startPowerWatch()
}

func Stop() {
	C.stopPowerWatch()
	setSend(nil)
}
