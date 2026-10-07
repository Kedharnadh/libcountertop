package visualview

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
)

type source struct{}

func (source) Kind() visual.Kind   { return visual.Orb }
func (source) Input() visual.Input { return visual.Input{} }
func (source) Hold() func()        { panic("audio acquired without a display helper") }
func (source) Loudest() float64    { return 0 }

func TestMissingHelperRemainsRetryable(t *testing.T) {
	attempts := 0
	d := missingDisplay{attempts: &attempts}
	w := New(source{}, Dependencies{GPU: gpu.New(d), Display: d})
	w.Keep(120, 80)
	w.Keep(120, 80)
	if attempts != 2 {
		t.Fatalf("open attempts=%d", attempts)
	}
}

type missingDisplay struct{ attempts *int }

func (d missingDisplay) Helper() *surface.Client         { *d.attempts++; return nil }
func (missingDisplay) Native() (int, int)                { return 120, 80 }
func (missingDisplay) Orientation() geometry.Orientation { return 0 }
