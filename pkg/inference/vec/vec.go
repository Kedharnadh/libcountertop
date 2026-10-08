package vec

func dotGo(a, b []float32) float32 {
	b = b[:len(a)]
	var s0, s1, s2, s3 float32
	i := 0
	for ; i+4 <= len(a); i += 4 {
		s0 += a[i] * b[i]
		s1 += a[i+1] * b[i+1]
		s2 += a[i+2] * b[i+2]
		s3 += a[i+3] * b[i+3]
	}
	s := s0 + s1 + s2 + s3
	for ; i < len(a); i++ {
		s += a[i] * b[i]
	}
	return s
}

func axpyGo(dst []float32, gain float32, x []float32) {
	dst = dst[:len(x)]
	for i, v := range x {
		dst[i] += gain * v
	}
}

func SumSquares(a []float32) float32 {
	var s0, s1, s2, s3 float32
	i := 0
	for ; i+4 <= len(a); i += 4 {
		s0 += a[i] * a[i]
		s1 += a[i+1] * a[i+1]
		s2 += a[i+2] * a[i+2]
		s3 += a[i+3] * a[i+3]
	}
	s := s0 + s1 + s2 + s3
	for ; i < len(a); i++ {
		s += a[i] * a[i]
	}
	return s
}
