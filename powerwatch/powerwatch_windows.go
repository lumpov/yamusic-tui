//go:build windows

package powerwatch

import (
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_WM_CLOSE                = 0x0010
	_WM_DESTROY              = 0x0002
	_WM_POWERBROADCAST       = 0x0218
	_WM_WTSSESSION_CHANGE    = 0x02B1
	_PBT_POWERSETTINGCHANGE  = 0x8013
	_WTS_SESSION_LOCK        = 0x7
	_WTS_SESSION_UNLOCK      = 0x8
	_NOTIFY_FOR_THIS_SESSION = 0
	_DEVICE_NOTIFY_WINDOW    = 0
	_HWND_MESSAGE            = ^uintptr(2)
	_POWER_MONITOR_OFF       = 0
	_POWER_MONITOR_ON        = 1
)

var guidSessionDisplayStatus = windows.GUID{
	Data1: 0x2B84C20E,
	Data2: 0xAD23,
	Data3: 0x4DDF,
	Data4: [8]byte{0x93, 0xDB, 0x05, 0xFF, 0xBD, 0x7E, 0xFC, 0xA5},
}

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modWtsapi32 = windows.NewLazySystemDLL("wtsapi32.dll")

	procGetModuleHandleW                   = modKernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW                   = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW                    = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow                      = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW                     = modUser32.NewProc("DefWindowProcW")
	procGetMessageW                        = modUser32.NewProc("GetMessageW")
	procTranslateMessage                   = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW                   = modUser32.NewProc("DispatchMessageW")
	procPostMessageW                       = modUser32.NewProc("PostMessageW")
	procRegisterPowerSettingNotification   = modUser32.NewProc("RegisterPowerSettingNotification")
	procUnregisterPowerSettingNotification = modUser32.NewProc("UnregisterPowerSettingNotification")
	procPostQuitMessage                    = modUser32.NewProc("PostQuitMessage")

	procWTSRegisterSessionNotification   = modWtsapi32.NewProc("WTSRegisterSessionNotification")
	procWTSUnRegisterSessionNotification = modWtsapi32.NewProc("WTSUnRegisterSessionNotification")
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type powerBroadcastSetting struct {
	PowerSetting windows.GUID
	DataLength   uint32
	Data         [4]byte
}

var (
	wndProcCallback = windows.NewCallback(powerWndProc)
	hwnd            atomic.Uintptr
	powerNotify     atomic.Uintptr
	locked          atomic.Bool
	started         atomic.Bool
)

func Supported() bool { return true }

func ScreenLocked() bool { return locked.Load() }

func Start(fn func(Event)) {
	setSend(fn)
	if started.CompareAndSwap(false, true) {
		go messageLoop()
	}
}

func Stop() {
	h := hwnd.Load()
	if h != 0 {
		procPostMessageW.Call(h, _WM_CLOSE, 0, 0)
	}
	setSend(nil)
}

func messageLoop() {
	className, err := windows.UTF16PtrFromString("YamusicTuiPowerWatch")
	if err != nil {
		started.Store(false)
		return
	}
	instance, _, _ := procGetModuleHandleW.Call(0)
	if instance == 0 {
		started.Store(false)
		return
	}
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndProcCallback,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	h, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, _HWND_MESSAGE, 0, instance, 0)
	if h == 0 {
		started.Store(false)
		return
	}
	hwnd.Store(h)
	notify, _, _ := procRegisterPowerSettingNotification.Call(h, uintptr(unsafe.Pointer(&guidSessionDisplayStatus)), _DEVICE_NOTIFY_WINDOW)
	powerNotify.Store(notify)
	procWTSRegisterSessionNotification.Call(h, _NOTIFY_FOR_THIS_SESSION)

	var msg struct {
		Hwnd    uintptr
		Message uint32
		Pad     uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		PtX     int32
		PtY     int32
		Pad2    uint32
	}
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	hwnd.Store(0)
	started.Store(false)
}

func powerWndProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case _WM_POWERBROADCAST:
		if wparam == _PBT_POWERSETTINGCHANGE && lparam != 0 {
			setting := (*powerBroadcastSetting)(unsafe.Pointer(lparam))
			if setting.PowerSetting == guidSessionDisplayStatus && setting.DataLength >= 4 {
				state := *(*uint32)(unsafe.Pointer(&setting.Data[0]))
				switch state {
				case _POWER_MONITOR_OFF:
					emit(DisplayOff)
				case _POWER_MONITOR_ON:
					emit(DisplayOn)
				}
			}
		}
		return 1
	case _WM_WTSSESSION_CHANGE:
		switch wparam {
		case _WTS_SESSION_LOCK:
			locked.Store(true)
			emit(SessionLocked)
		case _WTS_SESSION_UNLOCK:
			locked.Store(false)
			emit(SessionUnlocked)
		}
		return 0
	case _WM_CLOSE:
		n := powerNotify.Swap(0)
		if n != 0 {
			procUnregisterPowerSettingNotification.Call(n)
		}
		procWTSUnRegisterSessionNotification.Call(hwnd)
		procDestroyWindow.Call(hwnd)
		return 0
	case _WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wparam, lparam)
	return r
}
