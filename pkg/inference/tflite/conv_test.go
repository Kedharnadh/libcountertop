package tflite

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestConv2DOverManyChannelsMatchesADirectSum(t *testing.T) {
	const (
		ih, iw, ic = 5, 6, 32
		oc, kh, kw = 6, 3, 3
	)
	r := rand.New(rand.NewPCG(1, 2))
	fill := func(n int) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = float32(r.NormFloat64())
		}
		return out
	}
	x := tf([]int{1, ih, iw, ic}, fill(ih*iw*ic)...)
	w := tf([]int{oc, kh, kw, ic}, fill(oc*kh*kw*ic)...)
	bias := tf([]int{oc}, fill(oc)...)
	y := out()

	p := convParams{samePad: true, strideW: 1, strideH: 1, dilationW: 1, dilationH: 1}
	if err := kernels[OpConv2D](
		&OpDesc{Op: OpConv2D, conv: p},
		[]*Tensor{x, w, bias},
		[]*Tensor{y},
	); err != nil {
		t.Fatal(err)
	}
	shape(t, "conv", y.Shape, []int{1, ih, iw, oc})

	padH, padW := padFor(1, 1, ih, kh, ih), padFor(1, 1, iw, kw, iw)
	for oy := range ih {
		for ox := range iw {
			for f := range oc {
				want := float64(bias.F32[f])
				for ky := range kh {
					for kx := range kw {
						iy, ix := oy-padH+ky, ox-padW+kx
						if iy < 0 || iy >= ih || ix < 0 || ix >= iw {
							continue
						}
						for c := range ic {
							want += float64(
								x.F32[(iy*iw+ix)*ic+c],
							) * float64(
								w.F32[((f*kh+ky)*kw+kx)*ic+c],
							)
						}
					}
				}
				got := y.F32[(oy*iw+ox)*oc+f]
				if math.Abs(float64(got)-want) > 1e-3 {
					t.Fatalf("output (%d,%d) filter %d: %v, want %v", oy, ox, f, got, want)
				}
			}
		}
	}
}
