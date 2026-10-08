package castoutput

import (
	"fmt"

	"github.com/ygelfand/libcountertop/pkg/media/ownership"
	"github.com/ygelfand/libcountertop/pkg/media/pcm"
	"github.com/ygelfand/libcountertop/pkg/media/source"
)

type fakePlayer struct {
	owner *ownership.Owner[source.Source]
}

var testPlayer *fakePlayer

func newOutput() *Output {
	p := &fakePlayer{
		owner: ownership.New[source.Source](func(s source.Source) uint8 { return uint8(s.Kind()) }),
	}
	testPlayer = p
	return New(
		Options{
			Queue: &fakeQueue{},
			Media: func() Media { return p },
			Took:  func(pcm.Producer) {},
			Gave:  func(pcm.Producer) {},
		},
	)
}
func (p *fakePlayer) Began(s source.Source)   { p.owner.Begin(s) }
func (p *fakePlayer) Ended(s source.Source)   { p.owner.End(s) }
func (p *fakePlayer) Release(s source.Source) { p.owner.Release(s) }
func (p *fakePlayer) External(s source.Source) {
	if s == nil {
		p.owner.Clear()
	} else {
		p.owner.Set(s)
	}
}
func (p *fakePlayer) Changed() {}
func (p *fakePlayer) Now() source.Now {
	s, _ := p.owner.Get()
	if s == nil {
		return source.Now{}
	}
	return s.Now()
}
func (p *fakePlayer) Sourced() (string, bool) {
	s, ok := p.owner.Get()
	return fmt.Sprintf("%T", s), ok
}
