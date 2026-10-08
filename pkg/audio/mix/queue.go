package mix

import (
	"math"
	"sync"
)

type Source interface {
	Render(at uint64, out []int16)
}

type Tap interface{ Offer(mix []int16) }

type Queue struct {
	mu      sync.Mutex
	pending []int16
}

func (q *Queue) Push(samples []int16) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, samples...)
}

func (q *Queue) Overlay(samples []int16) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = Mix(q.pending, samples)
}

func (q *Queue) Take() []int16 {
	q.mu.Lock()
	defer q.mu.Unlock()

	pending := q.pending
	q.pending = nil
	return pending
}

func (q *Queue) Next(n int) []int16 {
	q.mu.Lock()
	defer q.mu.Unlock()

	n = min(len(q.pending), n)
	chunk := q.pending[:n]

	q.pending = q.pending[n:]
	if len(q.pending) == 0 {
		q.pending = nil
	}
	return chunk
}

func (q *Queue) Adjust(rewrite func([]int16)) {
	q.mu.Lock()
	defer q.mu.Unlock()
	rewrite(q.pending)
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = nil
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}

func Mix(into, add []int16) []int16 {
	for i, v := range add {
		if i >= len(into) {
			return append(into, add[i:]...)
		}
		into[i] = Clamp(int32(into[i]) + int32(v))
	}
	return into
}

func Clamp(v int32) int16 {
	switch {
	case v > math.MaxInt16:
		return math.MaxInt16
	case v < math.MinInt16:
		return math.MinInt16
	}
	return int16(v)
}
