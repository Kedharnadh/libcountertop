package component

import (
	"cmp"
	"context"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/runtime/service"
)

type Registry[T any] struct {
	eligible func(uint64) bool
	mu       sync.Mutex
	entries  []*entry
}

type entry struct {
	make  func() Component
	once  sync.Once
	c     Component
	phase Phase
	order int
	opts  []service.Option
	needs uint64
}

func (e *entry) resolve() { e.once.Do(func() { e.c = e.make() }) }

type Option func(*entry)

func Order(n int) Option { return func(e *entry) { e.order = n } }

func Supervise(opts ...service.Option) Option {
	return func(e *entry) { e.opts = append(e.opts, opts...) }
}

func Needs(capabilities uint64) Option { return func(e *entry) { e.needs |= capabilities } }

func New[T any](eligible func(uint64) bool) *Registry[T] { return &Registry[T]{eligible: eligible} }

func (r *Registry[T]) Add(p Phase, make func() Component, opts ...Option) {
	e := &entry{make: make, phase: p}
	for _, o := range opts {
		o(e)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

func Register[T any, C Component](r *Registry[T], p Phase, make func() C, opts ...Option) {
	r.Add(p, func() Component { return make() }, opts...)
}

func (r *Registry[T]) All() []Component {
	sorted := r.sorted()
	out := make([]Component, 0, len(sorted))
	for _, e := range sorted {
		out = append(out, e.c)
	}
	return out
}

func (r *Registry[T]) Entities() []esphome.Entity {
	var out []esphome.Entity
	for _, e := range r.sorted() {
		if ents, ok := e.c.(Entities); ok {
			out = append(out, ents.Entities()...)
		}
	}
	return out
}

func (r *Registry[T]) Actions() []*esphome.Action {
	var out []*esphome.Action
	for _, e := range r.sorted() {
		if acts, ok := e.c.(Actions); ok {
			out = append(out, acts.Actions()...)
		}
	}
	return out
}

func (r *Registry[T]) Handlers() []esphome.Handler {
	var out []esphome.Handler
	for _, e := range r.sorted() {
		if h, ok := e.c.(Handler); ok {
			out = append(out, h)
		}
	}
	return out
}

func (r *Registry[T]) Progress() []Progress {
	var out []Progress
	for _, e := range r.sorted() {
		s, ok := e.c.(Startup)
		if !ok {
			continue
		}

		p := s.Startup()
		p.Name = e.c.Name()
		out = append(out, p)
	}
	return out
}

func (r *Registry[T]) Ready() bool {
	for _, p := range r.Progress() {
		if !p.Settled() {
			return false
		}
	}
	return true
}

func (r *Registry[T]) Restore(c T) {
	for _, e := range r.sorted() {
		if v, ok := e.c.(Restorer[T]); ok {
			v.Restore(c)
		}
	}
}

func (r *Registry[T]) Group() *service.Group {
	g := service.New()
	r.AddTo(g)
	return g
}

func (r *Registry[T]) AddTo(g *service.Group) {
	for _, e := range r.sorted() {
		if svc, ok := e.c.(service.Service); ok {
			g.Add(svc, e.opts...)
			continue
		}

		_, starts := e.c.(Starter)
		_, closes := e.c.(Closer)
		if !starts && !closes {
			continue
		}
		g.Add(oneshot{e.c}, append([]service.Option{service.Once()}, e.opts...)...)
	}
}

// Embedding an interface promotes only that interface's methods.
type oneshot struct{ Component }

func (oneshot) Run(context.Context) error { return nil }

func (o oneshot) Start(ctx context.Context) error {
	if s, ok := o.Component.(Starter); ok {
		return s.Start(ctx)
	}
	return nil
}

func (o oneshot) Close() error {
	if c, ok := o.Component.(Closer); ok {
		return c.Close()
	}
	return nil
}

func (r *Registry[T]) sorted() []*entry {
	r.mu.Lock()
	out := slices.Clone(r.entries)
	r.mu.Unlock()

	if r.eligible != nil {
		out = slices.DeleteFunc(out, func(e *entry) bool { return !r.eligible(e.needs) })
	}
	for _, e := range out {
		e.resolve()
	}

	slices.SortStableFunc(out, func(a, b *entry) int {
		if v := cmp.Compare(a.phase, b.phase); v != 0 {
			return v
		}
		if v := cmp.Compare(a.order, b.order); v != 0 {
			return v
		}
		return cmp.Compare(a.c.Name(), b.c.Name())
	})
	return out
}
