package mediacard

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/media/nowplaying"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type testSource struct {
	now nowplaying.Now
	own bool
}

func (*testSource) Play()                 {}
func (*testSource) Pause()                {}
func (*testSource) Stop()                 {}
func (*testSource) Next()                 {}
func (*testSource) Previous()             {}
func (s *testSource) Now() nowplaying.Now { return s.now }
func (*testSource) Kind() nowplaying.Kind { return nowplaying.FromCast }
func (*testSource) Label() string         { return "" }
func (s *testSource) Open() bool          { return s.own }

type testPlayer struct{ source *testSource }

func (p testPlayer) Now() nowplaying.Now       { return p.source.Now() }
func (p testPlayer) Holder() nowplaying.Source { return p.source }
func (testPlayer) ResetSession()               {}

type testIdle struct{}

func (testIdle) Read() schema.Idle               { return schema.Idle{} }
func (testIdle) Update(func(*schema.Idle)) error { return nil }

func TestSourcePageReleasesTheCardAndCloseReleasesTheHold(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	src := &testSource{now: nowplaying.Now{Playing: true}}
	c := New(Dependencies{Player: testPlayer{src}, Idle: testIdle{}, Shell: stack})
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
