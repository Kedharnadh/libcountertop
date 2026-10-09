package idleview

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestRandomSpotMovesEveryThirtySeconds(t *testing.T) {
	v := &View{}
	c := schema.Idle{
		Placement: schema.Placement{Position: schema.PositionRandom, Align: schema.AlignRandom},
	}
	start := time.Now()
	first := v.Spot(c, start)
	if got := v.Spot(c, start.Add(moveEvery-time.Second)); got != first {
		t.Fatalf("moved early: %v then %v", first, got)
	}
	for n := 1; n <= 20; n++ {
		at := start.Add(time.Duration(n) * moveEvery)
		prev := v.spot
		if got := v.Spot(c, at); got == prev {
			t.Fatalf("step %d stayed at %v", n, got)
		}
	}
}

func TestOneRandomAxisKeepsTheOther(t *testing.T) {
	v := &View{}
	c := schema.Idle{
		Placement: schema.Placement{Position: schema.PositionTop, Align: schema.AlignRandom},
	}
	start := time.Now()
	for n := range 20 {
		got := v.Spot(c, start.Add(time.Duration(n)*moveEvery))
		if got.At != schema.PositionTop || got.Align == schema.AlignRandom {
			t.Fatalf("step %d: %v", n, got)
		}
	}
	c = schema.Idle{
		Placement: schema.Placement{Position: schema.PositionBottom, Align: schema.AlignLeft},
	}
	if got := v.Spot(c, start); got != (Spot{schema.PositionBottom, schema.AlignLeft}) {
		t.Fatalf("fixed spot became %v", got)
	}
}
