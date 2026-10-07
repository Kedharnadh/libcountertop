package a2dp

import (
	"github.com/ygelfand/libcountertop/pkg/media/pcm"
	mediasource "github.com/ygelfand/libcountertop/pkg/media/source"
)

type fakeQueue struct{ samples []int16 }

func (q *fakeQueue) Play(s []int16)         { q.samples = append(q.samples, s...) }
func (q *fakeQueue) Drain()                 { q.samples = nil }
func (q *fakeQueue) Queued() int            { return len(q.samples) / 2 }
func (q *fakeQueue) Adjust(f func([]int16)) { f(q.samples) }

type fakeMedia struct{}

func (fakeMedia) Began(mediasource.Source)   {}
func (fakeMedia) Ended(mediasource.Source)   {}
func (fakeMedia) Release(mediasource.Source) {}
func (fakeMedia) Changed()                   {}
func testSink() *Sink {
	return New(Options{Queue: &fakeQueue{}, Enabled: func() bool { return true }, Media: func() Media { return fakeMedia{} }, VolumeSet: func(int) {}, Took: func(pcm.Producer) {}, Gave: func(pcm.Producer) {}, DuckDB: func() float64 { return -15 }})
}
