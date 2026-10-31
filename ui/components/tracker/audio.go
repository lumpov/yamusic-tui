package tracker

import "time"

type audioPlayer interface {
	Play()
	Pause()
	Close() error
	SetVolume(float64)
	Volume() float64
	Seek(offset int64, whence int) (int64, error)
	IsPlaying() bool
	CurrentTime() (time.Duration, bool)
}

type audioBackend interface {
	Open(src *readWrapper) audioPlayer
}
