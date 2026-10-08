package castreceiver

import (
	"math"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/media/cast"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

type Volume interface {
	Level() int
	SetLevel(level int)
	Listen(changed func(level int)) (stop func())
}

type loudness struct {
	volume Volume

	mu  sync.Mutex
	svc *cast.Service

	wants chan int
}

func newLoudness(v Volume) *loudness {
	l := &loudness{volume: v, wants: make(chan int, 1)}
	safe.Go("cast volume", l.run)
	v.Listen(l.changed)
	return l
}

func (l *loudness) asked(level float64, muted bool) {
	want := int(math.Round(level * 100))
	if muted {
		want = 0
	}

	select {
	case <-l.wants:
	default:
	}
	l.wants <- want
}

func (l *loudness) run() {
	for want := range l.wants {
		l.volume.SetLevel(want)
	}
}

func (l *loudness) changed(level int) {
	l.mu.Lock()
	svc := l.svc
	l.mu.Unlock()

	if svc != nil {
		svc.Report(float64(level)/100, false)
	}
}

func (l *loudness) serve(svc *cast.Service) {
	l.mu.Lock()
	l.svc = svc
	l.mu.Unlock()

	if svc != nil {
		svc.Report(float64(l.volume.Level())/100, false)
	}
}
