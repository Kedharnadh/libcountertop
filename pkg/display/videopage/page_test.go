package videopage

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/media/source"
)

type level struct{ v int }

func (l *level) Level() int             { return l.v }
func (l *level) Set(v int)              { l.v = v }
func (l *level) Watch(func(int)) func() { return func() {} }

func page(t *testing.T, c Controls) *Page {
	t.Helper()
	stack := shell.New()
	t.Cleanup(stack.Close)
	return New(c, Dependencies{Shell: stack, Volume: &level{}})
}

type fake struct {
	now    source.Now
	paused int
}

func (f *fake) Play()              { f.now.Paused = false }
func (f *fake) Pause()             { f.now.Paused = true; f.paused++ }
func (f *fake) Stop()              {}
func (f *fake) Next()              {}
func (f *fake) Previous()          {}
func (f *fake) Now() source.Now    { return f.now }
func (f *fake) Kind() source.Kind  { return source.FromCast }
func (f *fake) Label() string      { return "phone" }
func (f *fake) Seek(time.Duration) {}
func (f *fake) CanSeek() bool      { return true }

func playing() *fake {
	return &fake{
		now: source.Now{
			Playing: true,
			Title:   "a video",
			Elapsed: 95 * time.Second,
			Length:  14 * time.Minute,
		},
	}
}

func TestFollowPointsTheControlsAtTheNextTrack(t *testing.T) {
	first, next := playing(), playing()
	p := page(t, first)
	p.Follow(next)
	p.Look().Controls.Pause()
	if first.paused != 0 || next.paused != 1 {
		t.Errorf("paused the first track %d times and the next %d", first.paused, next.paused)
	}
}

func TestRevealAndConcealToggleTheControls(t *testing.T) {
	p := page(t, playing())
	if p.Look().Shown {
		t.Fatal("controls showing before anything asked for them")
	}
	p.Reveal()
	if !p.Look().Shown {
		t.Fatal("reveal left the controls hidden")
	}
	p.Conceal()
	if p.Look().Shown {
		t.Error("conceal left the controls showing")
	}
}

func TestAPictureEndsTheLoadingSpinner(t *testing.T) {
	p := page(t, playing())
	p.SetPicture(true)
	if look := p.Look(); !look.Picture || look.Spinning {
		t.Errorf("picture %v, spinning %v", look.Picture, look.Spinning)
	}
}
