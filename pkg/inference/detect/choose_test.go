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

func (f *fakeSlots) Clear(n int) {
	f.cleared = append(f.cleared, n)
	delete(f.used, n)
}

func (f *fakeSlots) Loaded(n int) (wake.Model, bool) {
	id, ok := f.used[n]
	return wake.Model{ID: id}, ok
}

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
	if !slices.Equal(f.cleared, []int{0, 4}) {
		t.Errorf("cleared %v", f.cleared)
	}
}

func TestAWordThatWillNotLoadKeepsTheOneRunning(t *testing.T) {
	f := &fakeSlots{used: map[int]string{0: "a", 1: "b"}, refuse: "broken"}
	models := []wake.Model{{ID: "a"}, {ID: "b"}, {ID: "broken"}}

	accepted, loaded := Choose(f, models, []string{"broken", "missing"}, 2)

	if !slices.Equal(accepted, []string{"a", "b"}) {
		t.Errorf("accepted %q, want the words already running", accepted)
	}
	if !slices.Equal(loaded, []int{0, 1}) {
		t.Errorf("loaded %v, want [0 1]", loaded)
	}
	if len(f.cleared) > 0 {
		t.Errorf("cleared %v, want nothing", f.cleared)
	}
}
