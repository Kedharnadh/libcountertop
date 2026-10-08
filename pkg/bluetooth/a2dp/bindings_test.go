package a2dp

import (
	"github.com/ygelfand/libcountertop/pkg/audio/background"
	mediasource "github.com/ygelfand/libcountertop/pkg/media/source"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
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

type fakeSettings struct{}

func (fakeSettings) Read() schema.Bluetooth               { return schema.Bluetooth{Speaker: true} }
func (fakeSettings) Update(func(*schema.Bluetooth)) error { return nil }

type fakeDucking struct{}

func (fakeDucking) Read() schema.Media               { return schema.Media{DuckDB: -15} }
func (fakeDucking) Update(func(*schema.Media)) error { return nil }

type fakeVolume struct{}

func (fakeVolume) Level() int             { return 100 }
func (fakeVolume) Set(int)                {}
func (fakeVolume) Watch(func(int)) func() { return func() {} }
func testSink() *Sink {
	return New(
		Dependencies{
			Queue:       &fakeQueue{},
			Settings:    fakeSettings{},
			Media:       fakeMedia{},
			Volume:      fakeVolume{},
			Arbitration: &background.Arbiter{},
			Ducking:     fakeDucking{},
		},
	)
}
