package cameraview

import (
	"math"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

const unknownShape = 16.0 / 9.0

type row struct {
	n      int
	height float64
}

func Collage(shapes []float64, w, h, target, gap float64) [][]ui.Rect {
	if len(shapes) == 0 || w <= 0 || h <= 0 {
		return nil
	}
	target = max(1, min(target, h))
	rows := justify(shapes, w, target, gap)
	var pages [][]ui.Rect
	from, at := 0, 0
	for from < len(rows) {
		to, used := from+1, rows[from].height
		for to < len(rows) && used+gap+rows[to].height <= h {
			used += gap + rows[to].height
			to++
		}
		pages = append(pages, place(shapes[at:], rows[from:to], w, h, used, gap))
		for _, r := range rows[from:to] {
			at += r.n
		}
		from = to
	}
	return pages
}

func justify(shapes []float64, w, target, gap float64) []row {
	fill := func(sum float64, n int) float64 { return (w - gap*float64(n-1)) / sum }
	var rows []row
	sum, n := 0.0, 0
	for _, s := range shapes {
		before := 0.0
		if n > 0 {
			before = fill(sum, n)
		}
		sum, n = sum+s, n+1
		height := fill(sum, n)
		if height > target {
			continue
		}
		if n > 1 && before-target < target-height {
			rows = append(rows, row{n: n - 1, height: before})
			sum, n = s, 1
			if alone := fill(s, 1); alone <= target {
				rows = append(rows, row{n: 1, height: alone})
				sum, n = 0, 0
			}
			continue
		}
		rows = append(rows, row{n: n, height: height})
		sum, n = 0, 0
	}
	if n > 0 {
		rows = append(rows, row{n: n, height: min(target, fill(sum, n))})
	}
	return rows
}

func place(shapes []float64, rows []row, w, h, used, gap float64) []ui.Rect {
	scale := min(1, h/used)
	y := (h - used*scale) / 2
	var out []ui.Rect
	at := 0
	for _, r := range rows {
		span := gap * float64(r.n-1)
		for _, s := range shapes[at : at+r.n] {
			span += r.height * s
		}
		x := (w - span*scale) / 2
		for _, s := range shapes[at : at+r.n] {
			tw := r.height * s * scale
			out = append(out, ui.Rect{
				X: int(math.Round(x)),
				Y: int(math.Round(y)),
				W: int(math.Round(tw)),
				H: int(math.Round(r.height * scale)),
			})
			x += tw + gap*scale
		}
		y += (r.height + gap) * scale
		at += r.n
	}
	return out
}
