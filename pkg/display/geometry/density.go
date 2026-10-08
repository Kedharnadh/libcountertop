package geometry

import "math"

func DPI(w, h int, diagonal float64) float64 {
	if diagonal <= 0 || w <= 0 || h <= 0 {
		return 0
	}
	return math.Hypot(float64(w), float64(h)) / diagonal
}
