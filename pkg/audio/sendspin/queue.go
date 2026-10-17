package sendspin

const queueFrames = holdMax + maxSnap + Rate

type queue struct {
	ring []int16
	head int
	size int
}

func newQueue(frames int) queue { return queue{ring: make([]int16, frames*Channels)} }

func (q *queue) capacity() int { return len(q.ring) / Channels }

func (q *queue) frames() int { return q.size }

func (q *queue) room() int { return q.capacity() - q.size }

func (q *queue) drop(n int) {
	n = min(n, q.size)
	q.head = q.wrap(q.head + n)
	q.size -= n
}

func (q *queue) lead(n int) int {
	n = min(n, q.room())
	q.head = q.wrap(q.head - n)
	q.size += n
	q.clear(0, n)
	return n
}

func (q *queue) repeat() bool {
	if q.size == 0 || q.room() == 0 {
		return false
	}
	q.head = q.wrap(q.head - 1)
	q.size++
	next := q.wrap(q.head+1) * Channels
	copy(q.ring[q.head*Channels:q.head*Channels+Channels], q.ring[next:next+Channels])
	return true
}

func (q *queue) put(at int, samples []int16) int {
	n := min(len(samples)/Channels, q.capacity()-at)
	if n <= 0 {
		return 0
	}
	if at > q.size {
		q.clear(q.size, at-q.size)
	}
	q.span(at, n, func(ring []int16, done int) {
		copy(ring, samples[done*Channels:])
	})
	q.size = max(q.size, at+n)
	return n
}

func (q *queue) read(dst []int16) int {
	n := min(len(dst)/Channels, q.size)
	q.span(0, n, func(ring []int16, done int) {
		copy(dst[done*Channels:], ring)
	})
	return n * Channels
}

func (q *queue) reset() { q.head, q.size = 0, 0 }

func (q *queue) clear(at, n int) {
	q.span(at, n, func(ring []int16, _ int) { clear(ring) })
}

func (q *queue) span(at, n int, do func(ring []int16, done int)) {
	start := q.wrap(q.head + at)
	first := min(n, q.capacity()-start)
	do(q.ring[start*Channels:(start+first)*Channels], 0)
	if first < n {
		do(q.ring[:(n-first)*Channels], first)
	}
}

func (q *queue) wrap(i int) int {
	c := q.capacity()
	if c == 0 {
		return 0
	}
	return ((i % c) + c) % c
}
