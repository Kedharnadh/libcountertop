package detect

import (
	"errors"
	"slices"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/inference/wake"
)

type fakeSlots struct {
	used    map[int]string
	cleared []int
	refuse  string
}

func (f *fakeSlots) Use(n int, m wake.Model) error {
	if m.ID == f.refuse {
		return errors.New("refused")
	}
	f.used[n] = m.ID
	return nil
}

func (f *fakeSlots) Clear(n int) { f.cleared = append(f.cleared, n) }

func TestChooseKeepsEachWordInItsSlot(t *testing.T) {
	f := &fakeSlots{used: map[int]string{}, refuse: "broken"}
	models := []wake.Model{{ID: "a"}, {ID: "b"}, {ID: "broken"}}

	accepted, loaded := Choose(f, models, []string{"", "b", "missing", "broken"}, 5)

	if !slices.Equal(accepted, []string{"", "b", "", "", ""}) {
		t.Errorf("accepted %q", accepted)
	}
	if !slices.Equal(loaded, []int{1}) {
		t.Errorf("loaded %v, want [1]", loaded)
	}
	if !slices.Equal(f.cleared, []int{0, 2, 3, 4}) {
		t.Errorf("cleared %v", f.cleared)
	}
}
