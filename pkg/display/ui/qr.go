package ui

import (
	"image"
	"image/color"
	"image/draw"

	"rsc.io/qr"
)

func Code(text string, size int, dark, light color.Color) (*image.RGBA, error) {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, err
	}

	// The QR spec requires a quiet zone of four modules.
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
