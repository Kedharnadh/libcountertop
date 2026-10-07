package component

import (
	"sync"
	"sync/atomic"
	"testing"
)

type filtered struct{ name string }

func (f filtered) Name() string { return f.name }

func TestCapabilityFilterRunsBeforeConstruction(t *testing.T) {
	var builds atomic.Int32
	r := New[struct{}](func(needs uint64) bool { return needs&^uint64(1) == 0 })
	r.Add(Hardware, func() Component { builds.Add(1); return filtered{"panel"} }, Needs(2))
	r.Add(Hardware, func() Component { return filtered{"speaker"} }, Needs(1))
	if all := r.All(); len(all) != 1 || all[0].Name() != "speaker" || builds.Load() != 0 {
		t.Fatalf("filtered components=%v panel builds=%d", all, builds.Load())
	}
}

func TestConcurrentWalksConstructOnce(t *testing.T) {
	var builds atomic.Int32
	r := New[struct{}](nil)
	r.Add(Device, func() Component { builds.Add(1); return filtered{"one"} })
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() { r.All(); r.Progress(); r.Entities() })
	}
	workers.Wait()
	if builds.Load() != 1 {
		t.Fatalf("constructor ran %d times", builds.Load())
	}
}
