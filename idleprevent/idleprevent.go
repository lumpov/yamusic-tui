package idleprevent

import (
	"sync"

	"github.com/dece2183/yamusic-tui/log"
)

var (
	mu   sync.Mutex
	held bool
)

func Held() bool {
	mu.Lock()
	defer mu.Unlock()
	return held
}

func Set(on bool) {
	mu.Lock()
	defer mu.Unlock()
	if on == held {
		return
	}
	if on {
		if !acquire() {
			return
		}
		held = true
		log.Print(log.LVL_INFO, "idleprevent on")
		return
	}
	release()
	held = false
	log.Print(log.LVL_INFO, "idleprevent off")
}

func Stop() {
	Set(false)
}
