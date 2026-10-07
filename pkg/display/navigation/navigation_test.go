package navigation

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
)

type view struct{}

func (*view) Covers() bool { return true }

func TestFactoryCanRegisterOverlayAndConcurrentCallsResolveOnce(t *testing.T) {
	n := &Nav{}
	v := &view{}
	var calls atomic.Int32
	overlay := &Overlay{Priority: 2}
	n.For = func(v shell.View) *Screen { calls.Add(1); n.Show(overlay); return &Screen{View: v} }
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() { n.Showing([]shell.View{v}) })
	}
	workers.Wait()
	if calls.Load() != 1 || len(n.Overlays()) != 1 {
		t.Fatalf("factory calls=%d overlays=%d", calls.Load(), len(n.Overlays()))
	}
	n.Showing(nil)
	n.Showing([]shell.View{v})
	if calls.Load() != 2 {
		t.Fatal("removed view retained its cached screen")
	}
}

func TestOverlaySnapshotAndPriority(t *testing.T) {
	n := &Nav{}
	a, b := &Overlay{Priority: 2}, &Overlay{Priority: 1}
	n.Show(a)
	n.Show(b)
	n.Show(a)
	got := n.Overlays()
	if len(got) != 2 || got[0] != b || got[1] != a {
		t.Fatalf("overlays=%v", got)
	}
	got[0] = nil
	if n.Overlays()[0] != b {
		t.Fatal("caller changed overlay collection")
	}
}
