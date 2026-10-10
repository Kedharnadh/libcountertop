//go:build arm && !noasm

package vec

import "golang.org/x/sys/cpu"

var neon = cpu.ARM.HasNEON

// Taking the address of the first element of an empty slice panics.

func Dot(a, b []float32) float32 {
	if len(a) == 0 {
		return 0
	}
	b = b[:len(a)]
	if neon {
		return dotNEON(&a[0], &b[0], len(a))
	}
	return dotVFP(&a[0], &b[0], len(a))
}

func Dot4(x, w0, w1, w2, w3 []float32) (out [4]float32) {
	n := len(x)
	w0, w1, w2, w3 = w0[:n], w1[:n], w2[:n], w3[:n]
	m := n &^ 3
	if !neon || m == 0 {
		dot4Go(&out, x, w0, w1, w2, w3)
		return out
	}
	dot4NEON(&x[0], &w0[0], &w1[0], &w2[0], &w3[0], m, &out)
	if m < n {
		dot4Go(&out, x[m:], w0[m:], w1[m:], w2[m:], w3[m:])
	}
	return out
}

func AXPY(dst []float32, gain float32, x []float32) {
	if len(x) == 0 {
		return
	}
	dst = dst[:len(x)]
	axpyVFP(&dst[0], &x[0], len(x), gain)
}

//go:noescape
func dotVFP(a, b *float32, n int) float32

//go:noescape
func dotNEON(a, b *float32, n int) float32

//go:noescape
func dot4NEON(x, w0, w1, w2, w3 *float32, n int, out *[4]float32)

//go:noescape
func axpyVFP(dst, x *float32, n int, gain float32)
