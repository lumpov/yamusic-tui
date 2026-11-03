//go:build windows

package idleprevent

import (
	"runtime"
	"sync"

	"golang.org/x/sys/windows"
)

const (
	_ES_SYSTEM_REQUIRED = 0x00000001
	_ES_CONTINUOUS      = 0x80000000
)

var (
	modKernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procSetThreadExecutionState = modKernel32.NewProc("SetThreadExecutionState")
	loopOnce                    sync.Once
	cmdCh                       chan bool
)

func Supported() bool { return true }

func ensureLoop() {
	loopOnce.Do(func() {
		cmdCh = make(chan bool, 4)
		go func() {
			runtime.LockOSThread()
			for on := range cmdCh {
				if on {
					procSetThreadExecutionState.Call(uintptr(_ES_CONTINUOUS | _ES_SYSTEM_REQUIRED))
					continue
				}
				procSetThreadExecutionState.Call(uintptr(_ES_CONTINUOUS))
			}
		}()
	})
}

func acquire() bool {
	ensureLoop()
	cmdCh <- true
	return true
}

func release() {
	ensureLoop()
	cmdCh <- false
}
