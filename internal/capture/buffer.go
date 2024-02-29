package capture

import "time"

type Buffer []byte

func (buf Buffer) Length() time.Duration {
	return time.Millisecond * time.Duration(len(buf)) / pulseChannels / (pulseSampleRate / 1000) / (pulseBitSize / 8)
}
