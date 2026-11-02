package powerwatch

import "sync"

type Event int

const (
	DisplayOff Event = iota
	DisplayOn
	SessionLocked
	SessionUnlocked
)

var (
	sendMu sync.Mutex
	sendFn func(Event)
)

func emit(e Event) {
	sendMu.Lock()
	fn := sendFn
	sendMu.Unlock()
	if fn != nil {
		fn(e)
	}
}

func setSend(fn func(Event)) {
	sendMu.Lock()
	sendFn = fn
	sendMu.Unlock()
}
