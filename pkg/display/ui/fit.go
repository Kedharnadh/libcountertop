package ui

import (
	"image"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func DrawImageFit(s Surface, img image.Image, r Rect, on theme.Color) {
	if img == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	if w*r.H > h*r.W {
		h = h * r.W / w
		w = r.W
	} else {
		w = w * r.H / h
		h = r.H
	}
	DrawImageScaled(s, img, Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}, on)
}
