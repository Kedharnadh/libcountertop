package drawer

import (
	"slices"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/input/touch"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func called(s string) func() string { return func() string { return s } }

func TestEdgesRoundTripToTheTouchscreen(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range []schema.Edge{schema.EdgeLeft, schema.EdgeRight, schema.EdgeTop, schema.EdgeBottom} {
		got := AsTouch(e).String()
		if seen[got] {
			t.Errorf("two edges both map to %q", got)
		}
		seen[got] = true
	}
}

func TestTheOrderIsDeclaredNotTheOrderAdded(t *testing.T) {
	r := &Rail{}

	for _, e := range []Entry{
		{Name: called("Idle"), Order: OrderIdle},
		{Name: called("Apps"), Order: OrderApps},
		{Name: called("Settings"), Order: OrderSettings},
	} {
		r.Add(e)
	}

	want := []string{"Settings", "Apps", "Idle"}
	for i, e := range r.Entries() {
		if e.Label() != want[i] {
			t.Errorf("position %d is %s, want %s", i, e.Label(), want[i])
		}
	}
}

func TestAListIsAskedEachTimeAndSortedIn(t *testing.T) {
	r := &Rail{}
	r.Add(Entry{Name: called("Settings"), Order: OrderSettings})
	r.Add(Entry{Name: called("Idle"), Order: OrderIdle})
	docked := []string{"Player"}
	r.AddList(func() []Entry {
		var out []Entry
		for n, name := range docked {
			out = append(out, Entry{Name: called(name), Order: OrderFavorites + n})
		}
		return out
	})

	docked = []string{"Calls", "Player"}
	var got []string
	for _, e := range r.Entries() {
		got = append(got, e.Label())
	}
	want := []string{"Settings", "Calls", "Player", "Idle"}
	if !slices.Equal(got, want) {
		t.Errorf("entries %v, want %v", got, want)
	}
}

func TestEqualOrdersFallBackToTheName(t *testing.T) {
	first, second := &Rail{}, &Rail{}

	first.Add(Entry{Name: called("Zebra"), Order: 50})
	first.Add(Entry{Name: called("Aardvark"), Order: 50})

	second.Add(Entry{Name: called("Aardvark"), Order: 50})
	second.Add(Entry{Name: called("Zebra"), Order: 50})

	for i, e := range first.Entries() {
		if got := second.Entries()[i].Label(); got != e.Label() {
			t.Errorf("position %d is %s one way round and %s the other", i, e.Label(), got)
		}
	}
	if first.Entries()[0].Label() != "Aardvark" {
		t.Errorf("the tiebreak is not the name: %s came first", first.Entries()[0].Label())
	}
}

func railFor(hide *schema.Delay) (*Rail, *shell.Shell) {
	s := shell.New()
	return New(Options{
		Shell: s,
		Edge:  func() schema.Edge { return schema.EdgeRight },
		Hide:  func() schema.Delay { return *hide },
	}), s
}

func TestASwipeShowsTheDockForTheChosenTime(t *testing.T) {
	hide := schema.Delay15s
	r, s := railFor(&hide)
	defer s.Close()

	r.Gesture(touch.Gesture{Kind: touch.Swipe, From: touch.Left})
	if shown, _ := s.Dock(); shown {
		t.Fatal("a swipe from the other edge showed the dock")
	}
	r.Gesture(touch.Gesture{Kind: touch.Swipe, From: touch.Right})
	if shown, pinned := s.Dock(); !shown || pinned {
		t.Errorf("dock shown %v pinned %v, want shown for a while", shown, pinned)
	}
}

func TestNeverPinsTheDockAndATimeReleasesIt(t *testing.T) {
	hide := schema.DelayNever
	r, s := railFor(&hide)
	defer s.Close()

	r.Settle()
	if shown, pinned := s.Dock(); !shown || !pinned {
		t.Fatalf("never: dock shown %v pinned %v, want pinned", shown, pinned)
	}

	hide = schema.Delay15s
	r.Settle()
	if shown, _ := s.Dock(); shown {
		t.Error("choosing a time left the dock pinned")
	}
}

func TestTheDeclaredPlacesAreDistinct(t *testing.T) {
	seen := map[int]bool{}

	for _, at := range []int{OrderSettings, OrderApps, OrderFavorites, OrderIdle} {
		if seen[at] {
			t.Errorf("two things are declared at %d", at)
		}
		seen[at] = true
	}
}
