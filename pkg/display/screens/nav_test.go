package screens

import (
	"slices"
	"testing"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
)

type view struct{ covers bool }

func (v *view) Covers() bool { return v.covers }

func TestTheHighestCoveringViewIsTheBaseAndWhatIsAboveItFloats(t *testing.T) {
	under, page, card := &view{covers: true}, &view{covers: true}, &view{}
	n := &navigation.Nav{For: func(v shell.View) *Screen { return &Screen{View: v} }}
	covering, above := n.Showing([]shell.View{under, page, card})
	if covering == nil || covering.View != page {
		t.Fatalf("covering %v, want the page", covering)
	}
	if len(above) != 1 || above[0].View != card {
		t.Errorf("above %v, want the card", above)
	}
	if covering, _ := n.Showing([]shell.View{card}); covering != nil {
		t.Errorf("an overlay alone was treated as covering")
	}
}

func TestAViewNobodyDrawsIsLeftOut(t *testing.T) {
	known, unknown := &view{covers: true}, &view{}
	n := &navigation.Nav{For: func(v shell.View) *Screen {
		if v == known {
			return &Screen{View: v}
		}
		return nil
	}}
	covering, above := n.Showing([]shell.View{known, unknown})
	if covering == nil || len(above) != 0 {
		t.Errorf("covering %v above %v", covering, above)
	}
}

func TestTheTopmostOpaqueOverlayHidesEverythingBelowIt(t *testing.T) {
	off, on := func() bool { return false }, func() bool { return true }
	for _, tc := range []struct {
		name   string
		covers []func() bool
		want   int
	}{
		{"none", []func() bool{nil, off, nil}, -1},
		{"one", []func() bool{nil, on, nil}, 1},
		{"the higher of two", []func() bool{on, nil, on, nil}, 2},
		{"empty", nil, -1},
	} {
		overlays := make([]*navigation.Overlay, len(tc.covers))
		for i, c := range tc.covers {
			overlays[i] = &navigation.Overlay{Priority: i, Covers: c}
		}
		if got := navigation.Covering(overlays); got != tc.want {
			t.Errorf("%s: covering at %d, want %d", tc.name, got, tc.want)
		}
	}
}

type saver struct {
	view
	closed int
}

func (s *saver) Close() { s.closed++ }

func TestACoveredFrameBuildsOnlyWhatIsOnTop(t *testing.T) {
	sh := shell.New()
	idle := &saver{view: view{covers: true}}
	sh.Push(idle)
	var built, hidden []string
	builds := func(name string) func(*gogui.Window) gogui.View {
		return func(*gogui.Window) gogui.View {
			built = append(built, name)
			return blank()
		}
	}
	a := &App{o: Options{Shell: sh}}
	a.nav = &navigation.Nav{For: func(v shell.View) *Screen {
		return &Screen{View: v, Clear: true, Build: builds("saver")}
	}}
	for _, o := range []*navigation.Overlay{
		{Priority: 1, Build: builds("below"), Covered: func() { hidden = append(hidden, "below") }},
		{Priority: 2, Build: builds("full"), Covers: func() bool { return true }},
		{Priority: 3, Build: builds("above")},
	} {
		a.nav.Show(o)
	}

	a.root(nil)
	if !slices.Equal(built, []string{"full", "above"}) {
		t.Errorf("built %v, want only the covering overlay and what is above it", built)
	}
	if !slices.Equal(hidden, []string{"below"}) {
		t.Errorf("told %v it was covered, want the overlay below", hidden)
	}
	if idle.closed != 1 {
		t.Errorf("the saver's visual stopped %d times, want once", idle.closed)
	}
}

func TestOnlyAFastSweepFromTheRightEdgeGoesBack(t *testing.T) {
	const w = 1024
	cases := []struct {
		from [2]int
		x, y int
		want bool
	}{
		{[2]int{1000, 300}, 800, 310, true},
		{[2]int{600, 300}, 400, 300, false},
		{[2]int{1000, 300}, 950, 300, false},
		{[2]int{1000, 100}, 850, 400, false},
		{[2]int{960, 150}, 800, 150, false},
	}
	for _, c := range cases {
		if got := navigation.BackSweep(c.from, c.x, c.y, w); got != c.want {
			t.Errorf("from %v to %d,%d: %v, want %v", c.from, c.x, c.y, got, c.want)
		}
	}
}
