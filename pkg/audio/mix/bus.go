package mix

import (
	"sync"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Bus struct {
	mu    sync.Mutex
	lanes map[schema.Stream]*lane
	src   Source

	srcBuf []int16
}

type lane struct {
	Queue
	gain  float32
	muted bool
}

func (l *lane) level() float32 {
	if l.muted {
		return 0
	}
	return l.gain
}

func (b *Bus) lane(s schema.Stream) *lane {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held(s)
}

func (b *Bus) held(s schema.Stream) *lane {
	if b.lanes == nil {
		b.lanes = map[schema.Stream]*lane{}
	}
	l, ok := b.lanes[s]
	if !ok {
		l = &lane{gain: 1}
		b.lanes[s] = l
	}
	return l
}

func (b *Bus) Push(s schema.Stream, samples []int16) { b.lane(s).Push(samples) }

func (b *Bus) Overlay(s schema.Stream, samples []int16) { b.lane(s).Overlay(samples) }

func (b *Bus) Take(s schema.Stream) []int16 { return b.lane(s).Take() }

func (b *Bus) Adjust(s schema.Stream, rewrite func([]int16)) { b.lane(s).Adjust(rewrite) }

func (b *Bus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, l := range b.lanes {
		l.Clear()
	}
}

func (b *Bus) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	var most int
	for _, l := range b.lanes {
		most = max(most, l.Len())
	}
	return most
}

func (b *Bus) SetGain(s schema.Stream, gain float32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held(s).gain = max(gain, 0)
}

func (b *Bus) Gain(s schema.Stream) float32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held(s).level()
}

func (b *Bus) Mute(s schema.Stream, on bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held(s).muted = on
}

func (b *Bus) Muted(s schema.Stream) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held(s).muted
}

func (b *Bus) Attach(src Source) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.src = src
}

func (b *Bus) Next(at uint64, out []int32) (queued int, rendered bool) {
	clear(out)

	b.mu.Lock()
	for _, l := range b.lanes {
		chunk := l.Next(len(out))
		queued = max(queued, len(chunk))
		add(out, chunk, l.level())
	}
	src, media := b.src, b.held(schema.StreamMedia).level()
	b.mu.Unlock()

	if src == nil {
		return queued, false
	}
	if cap(b.srcBuf) < len(out) {
		b.srcBuf = make([]int16, len(out))
	}
	buf := b.srcBuf[:len(out)]
	clear(buf)
	src.Render(at, buf)
	add(out, buf, media)
	return queued, true
}

func add(into []int32, samples []int16, gain float32) {
	switch gain {
	case 0:
	case 1:
		for i, v := range samples {
			into[i] += int32(v)
		}
	default:
		for i, v := range samples {
			into[i] += int32(float32(v) * gain)
		}
	}
}
