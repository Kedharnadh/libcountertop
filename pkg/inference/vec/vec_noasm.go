//go:build (!arm64 && !arm) || noasm

package vec

func Dot(a, b []float32) float32 { return dotGo(a, b) }

func Dot4(x, w0, w1, w2, w3 []float32) (out [4]float32) {
	dot4Go(&out, x, w0, w1, w2, w3)
	return out
}

func AXPY(dst []float32, gain float32, x []float32) { axpyGo(dst, gain, x) }
