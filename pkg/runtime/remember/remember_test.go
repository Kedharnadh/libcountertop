package remember

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestSlotsSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open gave %v", err)
	}
	ids := NewSlot[[]string](s, "weather.dismissed")
	if got := ids.Get(); got != nil {
		t.Fatalf("empty slot = %v, want nil", got)
	}
	if err := ids.Set([]string{"a", "b"}); err != nil {
		t.Fatalf("Set gave %v", err)
	}
	if err := NewSlot[int](s, "other").Set(7); err != nil {
		t.Fatalf("Set gave %v", err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatalf("reopen gave %v", err)
	}
	if got := NewSlot[[]string](
		again,
		"weather.dismissed",
	).Get(); !slices.Equal(
		got,
		[]string{"a", "b"},
	) {
		t.Errorf("reopened slot = %v, want [a b]", got)
	}
	if got := NewSlot[int](again, "other").Get(); got != 7 {
		t.Errorf("reopened other = %d, want 7", got)
	}
}
