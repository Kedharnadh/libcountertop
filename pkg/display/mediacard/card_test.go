package mediacard

import (
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	media "github.com/ygelfand/libcountertop/pkg/media/source"
	"testing"
	"time"
)

type testSource struct {
	now media.Now
	own bool
}

func (*testSource) Play()            {}
func (*testSource) Pause()           {}
func (*testSource) Stop()            {}
func (*testSource) Next()            {}
func (*testSource) Previous()        {}
func (s *testSource) Now() media.Now { return s.now }
func (*testSource) Kind() media.Kind { return media.FromCast }
func (*testSource) Label() string    { return "" }
func (s *testSource) Open() bool     { return s.own }

func TestSourcePageReleasesTheCardAndCloseReleasesTheHold(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	src := &testSource{now: media.Now{Playing: true}}
	c := New(Options{Now: src.Now, Source: func() Source { return src }, Idle: func() time.Duration { return 0 }, Reset: func() {}, Shell: stack})
	c.Began(src)
	c.Follow()
	if stack.Top() != c.Page() {
		t.Fatal("new source did not open its card")
	}
	src.own = true
	c.Open()
	if stack.Open() {
		t.Fatal("source page left the old card held")
	}
	src.own = false
	c.Open()
	c.Close()
	if stack.Open() {
		t.Fatal("card remained held after Close")
	}
}
