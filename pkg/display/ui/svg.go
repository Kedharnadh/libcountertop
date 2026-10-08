package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/exp/shiny/iconvg"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

var vgMagic = []byte{0x89, 'I', 'V', 'G'}

func IsVG(icon Icon) bool { return bytes.HasPrefix(icon, vgMagic) }

var colorCache = map[iconKey]*image.RGBA{}

func inColor(icon Icon, w, h int) *image.RGBA {
	at := iconKey{icon: string(icon), w: w, h: h}

	iconMu.Lock()
	defer iconMu.Unlock()

	if img, ok := colorCache[at]; ok {
		return img
	}

	img, err := raster(icon, w, h)
	if err != nil {
		colorCache[at] = nil
		return nil
	}

	if iconBytes+len(img.Pix) > iconBudget {
		evict(at)
	}
	colorCache[at] = img
	iconBytes += len(img.Pix)
	return img
}

func DrawColored(s Surface, icon Icon, r Rect, on theme.Color) {
	side := min(r.W, r.H)
	if side <= 0 {
		return
	}

	img := inColor(icon, side, side)
	if img == nil {
		return
	}
	DrawImageScaled(
		s,
		img,
		Rect{X: r.X + (r.W-side)/2, Y: r.Y + (r.H-side)/2, W: side, H: side},
		on,
	)
}

func raster(icon Icon, w, h int) (*image.RGBA, error) {
	if IsVG(icon) {
		return paintVG(icon, w, h)
	}
	return paint(icon, w, h)
}

func paintVG(icon Icon, w, h int) (*image.RGBA, error) {
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	var z iconvg.Rasterizer
	z.SetDstImage(out, out.Bounds(), draw.Src)

	opts := iconvg.DecodeOptions{
		Palette: &iconvg.Palette{0: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
	}
	if err := iconvg.Decode(&z, icon, &opts); err != nil {
		return nil, fmt.Errorf("ui: decoding an iconvg icon: %w", err)
	}
	return out, nil
}

func paint(icon Icon, w, h int) (*image.RGBA, error) {
	svg, err := oksvg.ReadIconStream(bytes.NewReader(icon))
	if err != nil {
		return nil, fmt.Errorf("ui: reading an svg icon: %w", err)
	}
	if svg.ViewBox.W == 0 || svg.ViewBox.H == 0 {
		return nil, fmt.Errorf("ui: an svg icon with no view box")
	}

	side := min(float64(w), float64(h))
	scale := min(side/svg.ViewBox.W, side/svg.ViewBox.H)

	dw, dh := svg.ViewBox.W*scale, svg.ViewBox.H*scale
	svg.SetTarget((float64(w)-dw)/2, (float64(h)-dh)/2, dw, dh)

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	svg.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, out, out.Bounds())), 1)
	return out, nil
}
