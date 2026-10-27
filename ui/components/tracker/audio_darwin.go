//go:build darwin

package tracker

/*
#cgo LDFLAGS: -framework AudioToolbox -framework CoreFoundation
#include <AudioToolbox/AudioToolbox.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <time.h>

#define AQ_NUM_BUFFERS 3
#define AQ_BUFFER_SIZE 32768
#define AQ_MAX_PACKETS 512

extern void goAQStopped(uintptr_t handle);

typedef struct {
	uintptr_t goHandle;
	AudioFileStreamID stream;
	AudioQueueRef queue;
	AudioQueueBufferRef buffers[AQ_NUM_BUFFERS];
	int bufferInUse[AQ_NUM_BUFFERS];
	pthread_mutex_t mu;
	pthread_cond_t cv;
	int closed;
	int started;
	int wantPlay;
	int finishing;
	float volume;
	AudioFileTypeID fileType;
} AQPlayer;

static void queueCallback(void *ud, AudioQueueRef q, AudioQueueBufferRef buf) {
	AQPlayer *p = (AQPlayer *)ud;
	pthread_mutex_lock(&p->mu);
	for (int i = 0; i < AQ_NUM_BUFFERS; i++) {
		if (p->buffers[i] == buf) {
			p->bufferInUse[i] = 0;
			break;
		}
	}
	pthread_cond_broadcast(&p->cv);
	pthread_mutex_unlock(&p->mu);
}

static void runningListener(void *ud, AudioQueueRef q, AudioQueuePropertyID id) {
	AQPlayer *p = (AQPlayer *)ud;
	UInt32 running = 0;
	UInt32 size = sizeof(running);
	AudioQueueGetProperty(q, kAudioQueueProperty_IsRunning, &running, &size);
	int finishing = 0;
	pthread_mutex_lock(&p->mu);
	finishing = p->finishing;
	pthread_mutex_unlock(&p->mu);
	if (running == 0 && finishing) {
		goAQStopped(p->goHandle);
	}
}

static int waitFreeBuffer(AQPlayer *p) {
	pthread_mutex_lock(&p->mu);
	while (!p->closed) {
		for (int i = 0; i < AQ_NUM_BUFFERS; i++) {
			if (!p->bufferInUse[i] && p->buffers[i] != NULL) {
				int idx = i;
				pthread_mutex_unlock(&p->mu);
				return idx;
			}
		}
		struct timespec ts;
		clock_gettime(CLOCK_REALTIME, &ts);
		ts.tv_nsec += 50000000;
		if (ts.tv_nsec >= 1000000000L) {
			ts.tv_sec += 1;
			ts.tv_nsec -= 1000000000L;
		}
		pthread_cond_timedwait(&p->cv, &p->mu, &ts);
	}
	pthread_mutex_unlock(&p->mu);
	return -1;
}

static void packetsProc(void *ud, UInt32 numBytes, UInt32 numPackets, const void *data, AudioStreamPacketDescription *desc) {
	AQPlayer *p = (AQPlayer *)ud;
	if (p->queue == NULL || numBytes == 0) {
		return;
	}
	int idx = waitFreeBuffer(p);
	if (idx < 0) {
		return;
	}

	AudioQueueBufferRef buf = p->buffers[idx];
	if (numBytes > buf->mAudioDataBytesCapacity) {
		numBytes = buf->mAudioDataBytesCapacity;
	}
	memcpy(buf->mAudioData, data, numBytes);
	buf->mAudioDataByteSize = numBytes;
	if (desc && numPackets > 0) {
		UInt32 n = numPackets;
		if (n > buf->mPacketDescriptionCapacity) {
			n = buf->mPacketDescriptionCapacity;
		}
		memcpy(buf->mPacketDescriptions, desc, n * sizeof(AudioStreamPacketDescription));
		buf->mPacketDescriptionCount = n;
	} else {
		buf->mPacketDescriptionCount = 0;
	}

	pthread_mutex_lock(&p->mu);
	p->bufferInUse[idx] = 1;
	int wantStart = p->wantPlay && !p->started && !p->closed;
	if (wantStart) {
		p->started = 1;
	}
	pthread_mutex_unlock(&p->mu);

	if (buf->mPacketDescriptionCount > 0) {
		AudioQueueEnqueueBuffer(p->queue, buf, buf->mPacketDescriptionCount, buf->mPacketDescriptions);
	} else {
		AudioQueueEnqueueBuffer(p->queue, buf, 0, NULL);
	}
	if (wantStart) {
		AudioQueueStart(p->queue, NULL);
	}
}

static void propertyListener(void *ud, AudioFileStreamID stream, AudioFileStreamPropertyID prop, UInt32 *flags) {
	AQPlayer *p = (AQPlayer *)ud;
	if (prop != kAudioFileStreamProperty_ReadyToProducePackets) {
		return;
	}

	AudioStreamBasicDescription asbd;
	UInt32 size = sizeof(asbd);
	if (AudioFileStreamGetProperty(stream, kAudioFileStreamProperty_DataFormat, &size, &asbd) != noErr) {
		return;
	}

	pthread_mutex_lock(&p->mu);
	int hasQueue = p->queue != NULL;
	pthread_mutex_unlock(&p->mu);
	if (hasQueue) {
		AudioQueueReset(p->queue);
		return;
	}

	if (AudioQueueNewOutput(&asbd, queueCallback, p, NULL, NULL, 0, &p->queue) != noErr) {
		return;
	}

	UInt32 cookieSize = 0;
	Boolean writable = 0;
	if (AudioFileStreamGetPropertyInfo(stream, kAudioFileStreamProperty_MagicCookieData, &cookieSize, &writable) == noErr && cookieSize > 0) {
		void *cookie = malloc(cookieSize);
		if (cookie && AudioFileStreamGetProperty(stream, kAudioFileStreamProperty_MagicCookieData, &cookieSize, cookie) == noErr) {
			AudioQueueSetProperty(p->queue, kAudioQueueProperty_MagicCookie, cookie, cookieSize);
		}
		free(cookie);
	}

	AudioQueueSetParameter(p->queue, kAudioQueueParam_Volume, p->volume);
	AudioQueueAddPropertyListener(p->queue, kAudioQueueProperty_IsRunning, runningListener, p);

	for (int i = 0; i < AQ_NUM_BUFFERS; i++) {
		AudioQueueAllocateBufferWithPacketDescriptions(p->queue, AQ_BUFFER_SIZE, AQ_MAX_PACKETS, &p->buffers[i]);
		p->bufferInUse[i] = 0;
	}
}

static AQPlayer* aqCreate(uintptr_t goHandle, uint32_t fileType) {
	AQPlayer *p = (AQPlayer *)calloc(1, sizeof(AQPlayer));
	if (!p) {
		return NULL;
	}
	p->goHandle = goHandle;
	p->fileType = fileType;
	p->volume = 1;
	pthread_mutex_init(&p->mu, NULL);
	pthread_cond_init(&p->cv, NULL);
	AudioFileStreamOpen(p, propertyListener, packetsProc, fileType, &p->stream);
	return p;
}

static int aqParse(AQPlayer *p, const void *data, uint32_t size) {
	if (!p || p->closed || !p->stream || size == 0) {
		return -1;
	}
	return AudioFileStreamParseBytes(p->stream, size, data, 0);
}

static void aqPlay(AQPlayer *p) {
	if (!p || p->closed) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->wantPlay = 1;
	p->finishing = 0;
	AudioQueueRef q = p->queue;
	int started = p->started;
	if (q != NULL) {
		p->started = 1;
	}
	pthread_mutex_unlock(&p->mu);
	if (q != NULL) {
		AudioQueueStart(q, NULL);
		(void)started;
	}
}

static void aqPause(AQPlayer *p) {
	if (!p || p->closed) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->wantPlay = 0;
	AudioQueueRef q = p->queue;
	pthread_mutex_unlock(&p->mu);
	if (q != NULL) {
		AudioQueuePause(q);
	}
}

static void aqSetVolume(AQPlayer *p, float v) {
	if (!p) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->volume = v;
	AudioQueueRef q = p->queue;
	pthread_mutex_unlock(&p->mu);
	if (q != NULL) {
		AudioQueueSetParameter(q, kAudioQueueParam_Volume, v);
	}
}

static int aqHasQueue(AQPlayer *p) {
	if (!p) {
		return 0;
	}
	pthread_mutex_lock(&p->mu);
	int has = p->queue != NULL;
	pthread_mutex_unlock(&p->mu);
	return has;
}

static void aqFinish(AQPlayer *p) {
	if (!p || p->closed) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->finishing = 1;
	AudioQueueRef q = p->queue;
	pthread_mutex_unlock(&p->mu);
	if (q != NULL) {
		AudioQueueStop(q, 0);
	} else {
		goAQStopped(p->goHandle);
	}
}

static void aqSeekReset(AQPlayer *p) {
	if (!p || p->closed) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->finishing = 0;
	p->started = 0;
	for (int i = 0; i < AQ_NUM_BUFFERS; i++) {
		p->bufferInUse[i] = 0;
	}
	AudioQueueRef q = p->queue;
	AudioFileStreamID stream = p->stream;
	p->stream = NULL;
	pthread_cond_broadcast(&p->cv);
	pthread_mutex_unlock(&p->mu);

	if (q != NULL) {
		AudioQueueStop(q, 1);
		AudioQueueReset(q);
	}
	if (stream != NULL) {
		AudioFileStreamClose(stream);
	}
	AudioFileStreamOpen(p, propertyListener, packetsProc, p->fileType, &p->stream);
}

static void aqDestroy(AQPlayer *p) {
	if (!p) {
		return;
	}
	pthread_mutex_lock(&p->mu);
	p->closed = 1;
	p->wantPlay = 0;
	AudioQueueRef q = p->queue;
	AudioFileStreamID stream = p->stream;
	p->queue = NULL;
	p->stream = NULL;
	pthread_cond_broadcast(&p->cv);
	pthread_mutex_unlock(&p->mu);

	if (q != NULL) {
		AudioQueueRemovePropertyListener(q, kAudioQueueProperty_IsRunning, runningListener, p);
		AudioQueueStop(q, 1);
		AudioQueueDispose(q, 1);
	}
	if (stream != NULL) {
		AudioFileStreamClose(stream);
	}
	pthread_mutex_destroy(&p->mu);
	pthread_cond_destroy(&p->cv);
	free(p);
}
*/
import "C"
import (
	"io"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

var _ audioPlayer = (*darwinPlayer)(nil)

var darwinPlayers sync.Map

type darwinBackend struct{}

func newAudioBackend() audioBackend {
	return darwinBackend{}
}

func (darwinBackend) Open(src *readWrapper) audioPlayer {
	src.emitNextOnEOF = false
	return newDarwinPlayer(src)
}

type darwinPlayer struct {
	src      *readWrapper
	handle   cgo.Handle
	c        *C.AQPlayer
	mu       sync.Mutex
	ioMu     sync.Mutex
	playing  bool
	paused   bool
	closed   bool
	volume   float64
	stopCh   chan struct{}
	feedDone chan struct{}
	feedOnce sync.Once
}

func newDarwinPlayer(src *readWrapper) *darwinPlayer {
	p := &darwinPlayer{
		src:      src,
		volume:   1,
		paused:   true,
		stopCh:   make(chan struct{}),
		feedDone: make(chan struct{}),
	}
	p.handle = cgo.NewHandle(p)
	darwinPlayers.Store(uintptr(p.handle), p)
	p.c = C.aqCreate(C.uintptr_t(p.handle), C.uint32_t(C.kAudioFileMP3Type))
	return p
}

func (p *darwinPlayer) Play() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.paused = false
	p.playing = true
	p.mu.Unlock()
	C.aqPlay(p.c)
	p.feedOnce.Do(func() {
		go p.feed()
	})
}

func (p *darwinPlayer) Pause() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.paused = true
	p.mu.Unlock()
	C.aqPause(p.c)
}

func (p *darwinPlayer) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.playing = false
	p.paused = true
	close(p.stopCh)
	p.mu.Unlock()

	p.feedOnce.Do(func() {
		close(p.feedDone)
	})
	select {
	case <-p.feedDone:
	case <-time.After(500 * time.Millisecond):
	}

	if p.c != nil {
		C.aqDestroy(p.c)
		p.c = nil
	}
	darwinPlayers.Delete(uintptr(p.handle))
	p.handle.Delete()
	return nil
}

func (p *darwinPlayer) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.mu.Lock()
	p.volume = v
	closed := p.closed
	p.mu.Unlock()
	if closed || p.c == nil {
		return
	}
	C.aqSetVolume(p.c, C.float(v))
}

func (p *darwinPlayer) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *darwinPlayer) Seek(offset int64, whence int) (int64, error) {
	p.ioMu.Lock()
	defer p.ioMu.Unlock()
	if p.src == nil {
		return 0, io.EOF
	}
	pos, err := p.src.Seek(offset, whence)
	if p.c != nil {
		C.aqSeekReset(p.c)
	}
	return pos, err
}

func (p *darwinPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing && !p.paused
}

func (p *darwinPlayer) feed() {
	defer close(p.feedDone)
	buf := make([]byte, 16*1024)
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		p.mu.Lock()
		paused := p.paused
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		if paused {
			time.Sleep(20 * time.Millisecond)
			continue
		}

		p.ioMu.Lock()
		n, err := p.src.Read(buf)
		if n > 0 && p.c != nil {
			C.aqParse(p.c, unsafe.Pointer(&buf[0]), C.uint32_t(n))
		}
		p.ioMu.Unlock()

		if err == io.EOF {
			if p.c != nil {
				C.aqFinish(p.c)
			} else if p.src != nil {
				p.src.sendNext()
			}
			return
		}
		if n == 0 {
			time.Sleep(20 * time.Millisecond)
		}
	}
}

//export goAQStopped
func goAQStopped(handle C.uintptr_t) {
	v, ok := darwinPlayers.Load(uintptr(handle))
	if !ok {
		return
	}
	p, ok := v.(*darwinPlayer)
	if !ok || p == nil {
		return
	}
	p.mu.Lock()
	p.playing = false
	p.paused = true
	p.mu.Unlock()
	if p.src != nil {
		p.src.sendNext()
	}
}
