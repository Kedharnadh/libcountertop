package volume

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestClamp(t *testing.T) {
	for in, want := range map[int]int{-20: 0, 0: 0, 55: 55, 100: 100, 500: 100} {
		if got := Clamp(in); got != want {
			t.Errorf("Clamp(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestWatchHearsOnlyItsStream(t *testing.T) {
	var c Changes
	var heard []int
	stop := c.Watch(schema.Stream("media"), func(l int) { heard = append(heard, l) })

	c.Emit(Change{Stream: "media", Level: 10})
	c.Emit(Change{Stream: "alerts", Level: 20})
	stop()
	c.Emit(Change{Stream: "media", Level: 30})

	if len(heard) != 1 || heard[0] != 10 {
		t.Errorf("heard %v, want [10]", heard)
	}
}
