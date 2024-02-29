package capture

import (
	"github.com/mesilliac/pulse-simple"
	"github.com/relvacode/pulseclip/internal/notification"
	"log"
	"sync"
	"time"
)

const (
	pulseSampleRate    = 48000
	pulseChannels      = 2
	pulseSliceDuration = 250
	pulseBitSize       = 16
)

func New(basePath string, notifier notification.Notifier, bufferSize time.Duration) (*Recorder, error) {
	stream, err := pulse.Capture("PulseClip", "Pulse Clip Recording", &pulse.SampleSpec{
		Format:   pulse.SAMPLE_S16LE,
		Rate:     pulseSampleRate,
		Channels: pulseChannels,
	})
	if err != nil {
		return nil, err
	}

	var (
		sliceCount        = int(bufferSize.Milliseconds()) / pulseSliceDuration
		sliceBufferSize   = pulseSliceDuration * (pulseSampleRate / 1000) * pulseChannels * (pulseBitSize / 8)
		captureBufferSize = sliceBufferSize * sliceCount
	)
	return &Recorder{
		basePath:        basePath,
		stream:          stream,
		notifier:        notifier,
		signal:          make(chan Signal),
		done:            make(chan struct{}),
		sliceBufferSize: sliceBufferSize,
		sliceCount:      sliceCount,
		pool: sync.Pool{
			New: func() any { return make(Buffer, captureBufferSize) },
		},
	}, nil
}

type Recorder struct {
	basePath        string
	stream          *pulse.Stream
	notifier        notification.Notifier
	sliceBufferSize int
	sliceCount      int
	signal          chan Signal
	done            chan struct{}
	pool            sync.Pool
	clipPathLock    sync.Mutex
}

func (r *Recorder) save(buffer Buffer) {
	r.clipPathLock.Lock()
	defer r.clipPathLock.Unlock()

	defer r.pool.Put(buffer[:cap(buffer)]) // must reset length of buffer to capacity

	log.Printf("Saving new %s clip", buffer.Length())

	fullPath, err := writeClip(r.basePath, buffer)
	if err != nil {
		log.Printf("Error saving clip: %s", err)
		_ = r.notifier.Show("Failed to create audio clip", err.Error(), "")
		return
	}

	log.Printf("Clip saved to %s", fullPath)
	_ = r.notifier.Show("Audio Clip Saved", fullPath, "")
}

func (r *Recorder) Close() {
	_ = r.stream.Drain()
}

func (r *Recorder) Signal(sig Signal) {
	select {
	case <-r.done:
	case r.signal <- sig:
		log.Printf("Notify signal %T", sig)
	}
}

func (r *Recorder) Start() error {
	defer close(r.done)

	var (
		sliceIndex          int
		sliceBuffer         = make(Buffer, r.sliceBufferSize)
		clipRecordingBuffer = r.pool.Get().(Buffer)
	)
	for {
		_, err := r.stream.Read(sliceBuffer)
		if err != nil {
			return err
		}

		var recSlice = sliceIndex % r.sliceCount
		copy(clipRecordingBuffer[recSlice*r.sliceBufferSize:], sliceBuffer)
		sliceIndex++

		select {
		case sig := <-r.signal:
			switch sig.(type) {
			case Quit:
				return nil
			case Capture:
				var (
					capturedSlice = sliceIndex % r.sliceCount
					captureBuffer = r.pool.Get().(Buffer) // Get a new buffer to copy the current capture into
				)

				switch {
				case capturedSlice == 0:
					// Complete buffer
					copy(captureBuffer, clipRecordingBuffer)
				case sliceIndex < r.sliceCount:
					// Capture buffer not fully recorded yet
					n := copy(captureBuffer, clipRecordingBuffer[:sliceIndex*r.sliceBufferSize])
					captureBuffer = captureBuffer[:n]
				default:
					// Full buffer
					// Copy data from last pass
					n := copy(captureBuffer, clipRecordingBuffer[capturedSlice*r.sliceBufferSize:])
					copy(captureBuffer[n:], clipRecordingBuffer[:capturedSlice*r.sliceBufferSize])
				}

				// In a new routine encode and save the capture to disk
				go r.save(captureBuffer)
			}
		default:
		}
	}
}
