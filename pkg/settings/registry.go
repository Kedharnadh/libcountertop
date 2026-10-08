package setting

import (
	"slices"
	"strings"
	"sync"
)

type Bound interface {
	Domain() string
	Names() []string
	Value(name string) (string, error)
	Change(name, value string) error
}

type Registry struct {
	mu    sync.Mutex
	bound []Bound
}

func (r *Registry) Add(b ...Bound) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bound = append(r.bound, b...)
}

func (r *Registry) All() []Bound {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := slices.Clone(r.bound)
	slices.SortStableFunc(
		out,
		func(a, b Bound) int { return strings.Compare(a.Domain(), b.Domain()) },
	)
	return out
}
