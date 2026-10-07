package widgets

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func TestPaintedImageCacheAndPixelOrder(t *testing.T) {
	p := theme.Default()
	paints := 0
	draw := func(s ui.Surface, _ ui.Rect, _ theme.Theme) { paints++; s.Set(0, 0, theme.Color{R: 1, G: 2, B: 3}) }
	first := Painted(t.Name(), 2, 1, p.Background, p, draw)
	second := Painted(t.Name(), 2, 1, p.Background, p, draw)
	if first != second || paints != 1 {
		t.Fatalf("preview cache: first=%q second=%q paints=%d", first, second, paints)
	}
	img := ui.NewImage(1, 1, theme.Color{R: 1, G: 2, B: 3})
	pix := NRGBA(img)
	if len(pix) != 4 || pix[0] != 1 || pix[1] != 2 || pix[2] != 3 || pix[3] != 255 {
		t.Fatalf("pixels=%v", pix)
	}
}
