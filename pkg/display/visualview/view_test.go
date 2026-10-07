package visualview

import (
	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"testing"
)

type source struct{}

func (source) Kind() visual.Kind   { return visual.Orb }
func (source) Input() visual.Input { return visual.Input{} }
func (source) Hold() func()        { panic("audio acquired without a display helper") }
func (source) Loudest() float64    { return 0 }

func TestMissingHelperRemainsRetryable(t *testing.T) {
	attempts := 0
	w := New(source{}, Options{Open: func(int, int) (*gpu.Layer, error) { attempts++; return nil, gpu.ErrNoHelper }})
	w.Keep(120, 80)
	w.Keep(120, 80)
	if attempts != 2 {
		t.Fatalf("open attempts=%d", attempts)
	}
}
