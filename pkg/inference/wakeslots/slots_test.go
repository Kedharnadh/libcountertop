package wakeslots

import (
	"slices"
	"testing"
)

func TestSavedGapsMissingModelsAndExplicitDisableRemainPositional(t *testing.T) {
	for _, saved := range [][]string{{"", "B"}, {"missing", "B"}} {
		got := Initial(saved, []string{"A", "B"}, "A", 2)
		if !slices.Equal(got, []string{"", "B"}) {
			t.Fatal(got)
		}
		if id, ok := ID(got, 1); !ok || id != "B" {
			t.Fatal(id, ok)
		}
		if !slices.Equal(Advertised(got), []string{"B"}) {
			t.Fatal(Advertised(got))
		}
	}
	if got := Initial([]string{"", ""}, []string{"A"}, "A", 2); len(Advertised(got)) != 0 {
		t.Fatal("re-enabled disabled wake words", got)
	}
	if got := Initial(nil, []string{"A"}, "A", 2); got[0] != "A" {
		t.Fatal("fresh default missing", got)
	}
}
func TestSelectionDoesNotShareMutableInputOrSnapshots(t *testing.T) {
	var s Selection
	ids := []string{"", "B"}
	s.Set(ids)
	ids[1] = "A"
	got := s.Get()
	got[1] = "A"
	if id, ok := s.ID(1); !ok || id != "B" {
		t.Fatal(id, ok)
	}
}
