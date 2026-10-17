package sendspin

import "testing"

func BenchmarkRenderWithSecondsQueued(b *testing.B) {
	const period = 1024
	o := NewOutput(nil)
	o.queue = newQueue(queueFrames)
	o.ready, o.anchored, o.frame = true, true, 0
	chunk := tone(period, 1)
	buf := frames(period)
	o.write(0, tone(5*Rate, 1))

	from := uint64(0)
	b.ResetTimer()
	for b.Loop() {
		o.Render(from, buf)
		o.write(microsFor(int64(from)+5*Rate), chunk)
		from += period
	}
}
