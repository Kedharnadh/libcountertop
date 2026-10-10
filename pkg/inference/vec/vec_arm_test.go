//go:build arm && !noasm

package vec

import (
	"math"
	"strconv"
	"testing"
)

func TestNEONAndVFPAgree(t *testing.T) {
	if !neon {
		t.Skip("no NEON on this CPU")
	}
	for _, n := range lengths {
		if n == 0 {
			continue
		}
		a, b := random(n, uint64(n)+4000), random(n, uint64(n)+5000)

		var scale float64
		for i := range a {
			scale += math.Abs(float64(a[i]) * float64(b[i]))
		}

		vfp, simd := dotVFP(&a[0], &b[0], n), dotNEON(&a[0], &b[0], n)
		if d := math.Abs(float64(simd-vfp)) / math.Max(1, scale); d > 1e-6 {
			t.Errorf("n=%d: NEON %v, VFP %v, off by %.3g of what was added", n, simd, vfp, d)
		}
	}
}

func BenchmarkDotVFP(b *testing.B) {
	for _, n := range []int{32, 64, 512, 1024} {
		x, y := random(n, 1), random(n, 2)
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n * 4))
			for range b.N {
				sink = dotVFP(&x[0], &y[0], n)
			}
		})
	}
}
