// Package capture supplies bounded diagnostic capture, levels and PCM WAV writing.
package capture

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

func Duration(arg string, minimum, maximum time.Duration) (time.Duration, error) {
	seconds, err := strconv.ParseFloat(arg, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < minimum.Seconds() || seconds > maximum.Seconds() {
		return 0, fmt.Errorf("capture seconds must be between %g and %g", minimum.Seconds(), maximum.Seconds())
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

// Listen ends when the source closes, the duration expires or the command is canceled.
func Listen[T any](ctx context.Context, duration time.Duration, frames <-chan T, receive func(T)) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ctx.Err()
		case frame, ok := <-frames:
			if !ok {
				return ctx.Err()
			}
			receive(frame)
		}
	}
}

// Events owns copies of borrowed hook frames and closes admission before returning its snapshot.
func Events[T any](ctx context.Context, duration time.Duration, subscribe func(func(T)) func(), clone func(T) T) ([]T, error) {
	var mu sync.Mutex
	var frames []T
	closed := false
	stop := subscribe(func(frame T) {
		mu.Lock()
		defer mu.Unlock()
		if !closed {
			frames = append(frames, clone(frame))
		}
	})
	timer := time.NewTimer(duration)
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-timer.C:
	}
	timer.Stop()
	stop()
	mu.Lock()
	closed = true
	result := frames
	mu.Unlock()
	return result, err
}

type Levels struct {
	Count, Peak int
	Energy      float64
}

func Measure(samples []int16) Levels {
	s := Levels{Count: len(samples)}
	for _, sample := range samples {
		v := int(sample)
		s.Peak = max(s.Peak, v, -v)
		s.Energy += float64(v) * float64(v)
	}
	return s
}
func (s Levels) RMS() float64 {
	if s.Count == 0 {
		return 0
	}
	return math.Sqrt(s.Energy / float64(s.Count))
}
func DBFS(value float64) float64 { return math.Round(20*math.Log10(max(value, 1)/32768)*10) / 10 }
func (s Levels) String() string {
	return fmt.Sprintf("peakdbfs %.1f rmsdbfs %.1f", DBFS(float64(s.Peak)), DBFS(s.RMS()))
}

// Write saves complete interleaved signed 16-bit PCM frames at an explicit absolute path.
func Write(path string, samples []int16, rate, channels int) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("capture path must be absolute: %q", path)
	}
	if rate <= 0 || channels <= 0 || channels > 32767 || len(samples)%channels != 0 || uint64(rate) > (1<<32)-1 || uint64(rate)*uint64(channels)*2 > (1<<32)-1 || uint64(len(samples)) > ((1<<32)-37)/2 {
		return fmt.Errorf("invalid PCM geometry: rate=%d channels=%d samples=%d", rate, channels, len(samples))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := make([]byte, 44+len(samples)*2)
	copy(out, "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)-8))
	copy(out[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(out[16:], 16)
	binary.LittleEndian.PutUint16(out[20:], 1)
	binary.LittleEndian.PutUint16(out[22:], uint16(channels))
	binary.LittleEndian.PutUint32(out[24:], uint32(rate))
	binary.LittleEndian.PutUint32(out[28:], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(out[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(out[34:], 16)
	copy(out[36:], "data")
	binary.LittleEndian.PutUint32(out[40:], uint32(len(samples)*2))
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(out[44+2*i:], uint16(sample))
	}
	return os.WriteFile(path, out, 0o644)
}
