package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func Decode(b []byte) (*Image, error) {
	src, kind, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decoding %d bytes of artwork: %w", len(b), err)
	}

	box := src.Bounds()
	out := NewImage(box.Dx(), box.Dy(), theme.Color{})
	if out.w <= 0 || out.h <= 0 {
		return nil, fmt.Errorf("artwork is %dx%d %s", out.w, out.h, kind)
	}

	for y := range out.h {
		for x := range out.w {
			r, g, b, _ := src.At(box.Min.X+x, box.Min.Y+y).RGBA()
			out.Set(x, y, theme.Color{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)})
		}
	}
	return out, nil
}
