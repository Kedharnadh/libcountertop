package ui

import (
	"image"
	"image/color"
	"image/draw"

	"rsc.io/qr"
)

// Code renders text as a QR square of about size pixels, on its own quiet background.
//
// The module count is whatever the text needs, so the drawn size is rounded down to a whole number
// of pixels per module: a code whose modules are not all the same width is one a phone struggles to
// read.
func Code(text string, size int, dark, light color.Color) (*image.RGBA, error) {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, err
	}

	// Four modules of quiet zone, which the spec asks for and scanners rely on.
	const quiet = 4

	modules := c.Size + 2*quiet
	scale := max(size/modules, 1)
	side := modules * scale

	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Bounds(), image.NewUniform(light), image.Point{}, draw.Src)

	on := image.NewUniform(dark)
	for y := range c.Size {
		for x := range c.Size {
			if !c.Black(x, y) {
				continue
			}
			at := image.Rect(
				(x+quiet)*scale, (y+quiet)*scale,
				(x+quiet+1)*scale, (y+quiet+1)*scale)
			draw.Draw(out, at, on, image.Point{}, draw.Src)
		}
	}
	return out, nil
}
