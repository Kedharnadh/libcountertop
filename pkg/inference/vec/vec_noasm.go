//go:build (!arm64 && !arm) || noasm

package vec

func Dot(a, b []float32) float32 { return dotGo(a, b) }

func AXPY(dst []float32, gain float32, x []float32) { axpyGo(dst, gain, x) }
