//go:build windows && !nomedia

package win

import (
	"math"
	"syscall"
	"unsafe"

	winrtplayback "github.com/dece2183/media-winrt-go/windows/media/playback"
	"github.com/go-ole/go-ole"
)

type iMediaPlayerVtbl struct {
	ole.IInspectableVtbl
	GetAutoPlay          uintptr
	SetAutoPlay          uintptr
	GetNaturalDuration   uintptr
	GetPosition          uintptr
	SetPosition          uintptr
	GetBufferingProgress uintptr
	GetCurrentState      uintptr
	GetCanSeek           uintptr
	GetCanPause          uintptr
	GetIsLoopingEnabled  uintptr
	SetIsLoopingEnabled  uintptr
	GetIsProtected       uintptr
	GetIsMuted           uintptr
	SetIsMuted           uintptr
	GetPlaybackRate      uintptr
	SetPlaybackRate      uintptr
	GetVolume            uintptr
	SetVolume            uintptr
}

func setMediaPlayerVolume(player *winrtplayback.MediaPlayer, vol float64) error {
	itf := player.MustQueryInterface(ole.NewGUID(winrtplayback.GUIDiMediaPlayer))
	defer itf.Release()
	insp := (*ole.IInspectable)(unsafe.Pointer(itf))
	vtbl := (*iMediaPlayerVtbl)(unsafe.Pointer(insp.RawVTable))
	hr, _, _ := syscall.SyscallN(
		vtbl.SetVolume,
		uintptr(unsafe.Pointer(insp)),
		uintptr(math.Float64bits(vol)),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}
