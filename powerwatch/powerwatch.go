package powerwatch

import (
	"fmt"
	"sync"

	"github.com/dece2183/yamusic-tui/log"
)

type Event int

const (
	DisplayOff Event = iota
	DisplayOn
	SessionLocked
	SessionUnlocked
)

func (e Event) String() string {
	switch e {
	case DisplayOff:
		return "display-off"
	case DisplayOn:
		return "display-on"
	case SessionLocked:
		return "session-locked"
	case SessionUnlocked:
		return "session-unlocked"
	default:
		return fmt.Sprintf("event(%d)", int(e))
	}
}

var (
	sendMu sync.Mutex
	sendFn func(Event)
)

func emit(e Event) {
	sendMu.Lock()
	fn := sendFn
	sendMu.Unlock()
	log.Print(log.LVL_INFO, "powerwatch %s listener=%v", e, fn != nil)
	if fn != nil {
		fn(e)
	}
}

func setSend(fn func(Event)) {
	sendMu.Lock()
	sendFn = fn
	sendMu.Unlock()
	if fn != nil {
		log.Print(log.LVL_INFO, "powerwatch listen")
	} else {
		log.Print(log.LVL_INFO, "powerwatch stop")
	}
}
