//go:build arm64 && !noasm

package vec

// Taking the address of the first element of an empty slice panics.

func Dot(a, b []float32) float32 {
	if len(a) == 0 {
		return 0
	}
	b = b[:len(a)]
	return dotNEON(&a[0], &b[0], len(a))
}

func Dot4(x, w0, w1, w2, w3 []float32) (out [4]float32) {
	dot4Go(&out, x, w0, w1, w2, w3)
	return out
}

func AXPY(dst []float32, gain float32, x []float32) {
	if len(x) == 0 {
		return
	}
	dst = dst[:len(x)]
	axpyNEON(&dst[0], &x[0], len(x), gain)
}

//go:noescape
func dotNEON(a, b *float32, n int) float32

//go:noescape
func axpyNEON(dst, x *float32, n int, gain float32)
