package widgets

import "testing"

func TestSunFractionWrapsMidnight(t *testing.T) {
	rise, _ := minutes("06:30")
	set, ok := minutes("19:30")
	if !ok || rise != 390 || set != 1170 {
		t.Fatalf("minutes %d %d %v", rise, set, ok)
	}
	for _, c := range []struct {
		now  int
		day  bool
		want float32
	}{
		{390, true, 0},
		{780, true, 0.5},
		{1170, true, 1},
		{1170, false, 0},
		{60, false, 0.5},
		{390, false, 1},
	} {
		if got := sunFraction(rise, set, c.now, c.day); got != c.want {
			t.Errorf("sunFraction(%d, %v) = %v, want %v", c.now, c.day, got, c.want)
		}
	}
	if _, ok := minutes("noon"); ok {
		t.Error("a word read as a time")
	}
}
