package tracker

type audioPlayer interface {
	Play()
	Pause()
	Close() error
	SetVolume(float64)
	Volume() float64
	Seek(offset int64, whence int) (int64, error)
	IsPlaying() bool
}

type audioBackend interface {
	Open(src *readWrapper) audioPlayer
}
