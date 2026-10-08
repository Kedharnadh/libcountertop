package detect

import (
	"slices"
	"sync/atomic"
)

type Queue struct {
	frames  chan []int16
	dropped atomic.Uint64
}

func NewQueue(capacity int) *Queue {
	return &Queue{frames: make(chan []int16, max(capacity, 1))}
}

func (q *Queue) Feed(samples []int16) {
	select {
	case q.frames <- slices.Clone(samples):
	default:
		q.dropped.Add(1)
	}
}
func (q *Queue) Listen(string) (<-chan []int16, func()) { return q.frames, func() {} }
func (q *Queue) Dropped() uint64                        { return q.dropped.Load() }
