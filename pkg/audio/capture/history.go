package capture

import (
	"sync"
	"time"
)

type History struct {
	rate int

	mu     sync.Mutex
	buf    []int16
	at     int
	filled bool
}

func NewHistory(rate int, span time.Duration) *History {
	return &History{rate: rate, buf: make([]int16, max(samplesIn(span, rate), 1))}
}

func samplesIn(d time.Duration, rate int) int { return int(d/time.Millisecond) * rate / 1000 }

func (h *History) Add(frame []int16) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, s := range frame {
		h.buf[h.at] = s
		h.at++
		if h.at == len(h.buf) {
			h.at, h.filled = 0, true
		}
	}
}

func (h *History) Recent(d time.Duration) []int16 {
	want := samplesIn(d, h.rate)

	h.mu.Lock()
	defer h.mu.Unlock()

	have := len(h.buf)
	if !h.filled {
		have = h.at
	}
	want = min(want, have)

	out := make([]int16, 0, want)
	start := h.at - want
	if start < 0 {
		start += len(h.buf)
	}
	for i := range want {
		out = append(out, h.buf[(start+i)%len(h.buf)])
	}
	return out
}
