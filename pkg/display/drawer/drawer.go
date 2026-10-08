package drawer

import (
	"cmp"
	"slices"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/input/touch"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	OrderSettings = 30
	OrderCalls    = 35
	OrderPlayer   = 40
	OrderIdle     = 50
)

type Entry struct {
	Name func() string

	Order int

	Glyph func() string

	Open func()
}

func (e Entry) Label() string {
	if e.Name == nil {
		return ""
	}
	return e.Name()
}

type Options struct {
	Shell *shell.Shell
	Edge  func() schema.Edge
}

type Rail struct {
	opts Options

	mu      sync.Mutex
	entries []Entry
}

func New(opts Options) *Rail { return &Rail{opts: opts} }

func (r *Rail) Name() string { return "drawer" }

func (r *Rail) Add(e Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

func (r *Rail) Entries() []Entry {
	r.mu.Lock()
	out := append([]Entry(nil), r.entries...)
	r.mu.Unlock()

	slices.SortStableFunc(out, func(a, b Entry) int {
		if v := cmp.Compare(a.Order, b.Order); v != 0 {
			return v
		}
		return cmp.Compare(a.Label(), b.Label())
	})
	return out
}

func (r *Rail) Gesture(g touch.Gesture) {
	s := r.opts.Shell
	if s.Open() || s.Closing() {
		return
	}
	if g.Kind == touch.Swipe && g.From == AsTouch(r.opts.Edge()) {
		s.Push(r)
	}
}

func (r *Rail) Covers() bool { return false }

func AsTouch(e schema.Edge) touch.Edge {
	switch e {
	case schema.EdgeLeft:
		return touch.Left
	case schema.EdgeTop:
		return touch.Top
	case schema.EdgeBottom:
		return touch.Bottom
	}
	return touch.Right
}
