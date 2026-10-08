package widgets

import "testing"

func TestCoverKeysStopAtTheEnds(t *testing.T) {
	open := coverKeys(
		dressed("cover.blind", "open", map[string]any{"current_position": 100.0}, nil),
	)
	if !open[0].disabled || open[2].disabled {
		t.Errorf("a fully open blind: %+v", open)
	}
	closing := coverKeys(
		dressed("cover.blind", "closing", map[string]any{"current_position": 100.0}, nil),
	)
	if closing[0].disabled {
		t.Error("a closing blind cannot be opened again")
	}
	gate := coverKeys(dressed("cover.gate", "closed", map[string]any{"device_class": "gate"}, nil))
	if gate[0].glyph != 0xF084E || !gate[2].disabled {
		t.Errorf("a closed gate: %+v", gate)
	}
}

func TestOptionsWrapAround(t *testing.T) {
	sel := dressed("select.mode", "c", map[string]any{"options": []any{"a", "b", "c"}}, nil)
	if nextOption(sel, 1) != "a" || nextOption(sel, -1) != "b" {
		t.Errorf("next %q previous %q", nextOption(sel, 1), nextOption(sel, -1))
	}
	if keys := optionKeys(
		dressed("select.one", "a", map[string]any{"options": []any{"a"}}, nil),
	); !keys[0].disabled {
		t.Error("one option still steps")
	}
}

func TestStepsSnapToTheGrid(t *testing.T) {
	for _, tt := range []struct {
		current, step, lo, hi float64
		dir                   int
		want                  float64
	}{
		{70, 1, 70, 90, 1, 71},
		{70, 1, 70, 90, -1, 70},
		{89.5, 1, 70, 90, 1, 90},
		{71.2, 0.5, 70, 90, 1, 71.5},
	} {
		if got := stepValue(tt.current, tt.step, tt.lo, tt.hi, tt.dir); got != tt.want {
			t.Errorf(
				"stepValue(%v, %v, %v, %v, %d) = %v, want %v",
				tt.current,
				tt.step,
				tt.lo,
				tt.hi,
				tt.dir,
				got,
				tt.want,
			)
		}
	}
}
