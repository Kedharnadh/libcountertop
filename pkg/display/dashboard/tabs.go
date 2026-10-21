package dashboard

import (
	"cmp"
	"slices"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/hook"
)

type Tab struct {
	Kind string
	Key  string
	Name string
}

type source struct {
	order int
	list  func() []Tab
}

type Tabs struct {
	Changed hook.Hook[struct{}]

	mu      sync.Mutex
	sources []source
	showing string
	redraw  func()
}

func NewTabs(redraw func()) *Tabs { return &Tabs{redraw: redraw} }

func (t *Tabs) Add(order int, list func() []Tab) {
	t.mu.Lock()
	t.sources = append(t.sources, source{order, list})
	slices.SortStableFunc(t.sources, func(a, b source) int { return cmp.Compare(a.order, b.order) })
	t.mu.Unlock()
}

func (t *Tabs) List() []Tab {
	t.mu.Lock()
	sources := slices.Clone(t.sources)
	t.mu.Unlock()
	var out []Tab
	for _, s := range sources {
		out = append(out, s.list()...)
	}
	return out
}

func (t *Tabs) Showing() (Tab, bool) {
	t.mu.Lock()
	key := t.showing
	t.mu.Unlock()
	if key == "" {
		return Tab{}, false
	}
	for _, tab := range t.List() {
		if tab.Key == key {
			return tab, true
		}
	}
	return Tab{}, false
}

func (t *Tabs) Show(key string) {
	t.mu.Lock()
	if t.showing == key {
		key = ""
	}
	t.showing = key
	t.mu.Unlock()
	t.changed()
}

func (t *Tabs) Clock() {
	t.mu.Lock()
	changed := t.showing != ""
	t.showing = ""
	t.mu.Unlock()
	if changed {
		t.changed()
	}
}

func (t *Tabs) changed() {
	t.Changed.Emit(struct{}{})
	if t.redraw != nil {
		t.redraw()
	}
}
