package a2dp

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/bluetooth/avrcp"
)

func list() []avrcp.Item {
	return []avrcp.Item{
		{UID: 11, Track: avrcp.Track{Title: "Black Cow"}},
		{UID: 12, Track: avrcp.Track{Title: "Aja"}},
		{UID: 13, Track: avrcp.Track{Title: "Deacon Blues"}},
	}
}

func titles(items []avrcp.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Track.Title)
	}
	return out
}

func same(t *testing.T, got []avrcp.Item, want ...string) {
	t.Helper()

	names := titles(got)
	if len(names) != len(want) {
		t.Fatalf("up next is %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("up next is %v, want %v", names, want)
		}
	}
}

func TestANewListingForgetsTheOldHandles(t *testing.T) {
	s := testSink()

	s.holds(list())
	s.mu.Lock()
	s.handles = map[uint64]string{11: "2964601"}
	s.mu.Unlock()

	s.holds(list())

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.handles) != 0 {
		t.Errorf("a handle survived the listing it was learned for: %v", s.handles)
	}
}

func TestTheUidSaysWhereInTheListWeAre(t *testing.T) {
	state := avrcp.State{UID: 12, Track: avrcp.Track{Title: "Aja"}}
	same(t, after(list(), state), "Deacon Blues")
}

func TestWithoutAUidTheTitleIsWhatMatches(t *testing.T) {
	state := avrcp.State{Track: avrcp.Track{Title: "Aja"}}
	same(t, after(list(), state), "Deacon Blues")
}

func TestNothingPlayingLeavesTheWholeList(t *testing.T) {
	state := avrcp.State{UID: nothing}
	same(t, after(list(), state), "Black Cow", "Aja", "Deacon Blues")
}

func TestATrackThatIsNotInTheListLeavesItWhole(t *testing.T) {
	state := avrcp.State{UID: 99, Track: avrcp.Track{Title: "Peg"}}
	same(t, after(list(), state), "Black Cow", "Aja", "Deacon Blues")
}

func TestTheEndOfTheListIsEmpty(t *testing.T) {
	state := avrcp.State{UID: 13, Track: avrcp.Track{Title: "Deacon Blues"}}
	same(t, after(list(), state))
}
