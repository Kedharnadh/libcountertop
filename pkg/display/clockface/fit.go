package clockface

import "github.com/ygelfand/libcountertop/pkg/display/ui"

func fit(weight ui.Weight, text string, maxW, maxH int) int {
	size := max(maxH, 1)

	for range 8 {
		w, _ := ui.MustLoad(weight, size).Measure(text)
		if w <= maxW || size <= 1 {
			return size
		}

		next := size * maxW / w
		if next >= size {
			next = size - 1
		}
		size = max(next, 1)
	}
	return size
}
