package setting

import (
	"errors"
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

type Snapshot map[string]map[string]string

type portable interface {
	Export() map[string]string
	Import(map[string]string) (int, error)
}

func (r *Registry) Export() Snapshot {
	out := Snapshot{}
	for _, b := range r.All() {
		p, ok := b.(portable)
		if !ok {
			continue
		}
		for name, value := range p.Export() {
			if out[b.Domain()] == nil {
				out[b.Domain()] = map[string]string{}
			}
			out[b.Domain()][name] = value
		}
	}
	return out
}

func (r *Registry) Import(s Snapshot) (int, error) {
	var (
		applied int
		errs    []error
	)
	for _, b := range r.All() {
		p, ok := b.(portable)
		values := s[b.Domain()]
		if !ok || len(values) == 0 {
			continue
		}
		n, err := p.Import(values)
		applied += n
		if err != nil {
			errs = append(errs, err)
		}
	}
	return applied, errors.Join(errs...)
}
