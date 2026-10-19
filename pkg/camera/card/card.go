package card

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

var (
	background = theme.Color{R: 0x18, G: 0x18, B: 0x18}
	ink        = theme.Color{R: 0xf0, G: 0xf0, B: 0xf0}
)

func Draw(w, h int, icon, text string) []byte {
	img := ui.NewImage(w, h, background)
	short := min(w, h)
	side := short / 4
	glyph(img, icons.Named(icon), (w-side)/2, h/2-side, side)
	f, err := ui.Load(ui.Medium, max(short/16, 10))
	if err == nil {
		ui.DrawTextIn(img, f, ui.Rect{Y: h/2 + short/24, W: w, H: short / 8}, ink, background, text)
	}
	return luma(img)
}

func glyph(img *ui.Image, r rune, x, y, side int) {
	parsed, err := opentype.Parse(icons.MDI)
	if err != nil {
		return
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(side), DPI: 72})
	if err != nil {
		return
	}
	defer face.Close()
	mask := image.NewAlpha(image.Rect(0, 0, side, side))
	d := font.Drawer{Dst: mask, Src: image.Opaque, Face: face, Dot: fixed.P(0, side*7/8)}
	d.DrawString(icons.Glyph(r))
	for iy := range side {
		for ix := range side {
			if a := mask.AlphaAt(ix, iy).A; a > 0 {
				img.Set(x+ix, y+iy, background.Mix(ink, a))
			}
		}
	}
}

func luma(img *ui.Image) []byte {
	w, h := img.Size()
	out := make([]byte, w*h)
	for y := range h {
		for x := range w {
			c := img.At(x, y)
			lum := (299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000
			out[y*w+x] = byte(16 + lum*219/255)
		}
	}
	return out
}
