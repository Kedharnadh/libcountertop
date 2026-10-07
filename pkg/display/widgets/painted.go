package widgets

import (
	"fmt"

	gui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Paint func(ui.Surface, ui.Rect, theme.Theme)

func Painted(key string, w, h int, fill theme.Color, palette theme.Theme, draw Paint) string {
	key = fmt.Sprintf("%s/%s/%dx%d", key, palette.Name, w, h)
	if gui.HasImage(key) {
		return "mem:" + key
	}
	img := ui.NewImage(w, h, fill)
	draw(img, ui.Rect{W: w, H: h}, palette)
	return gui.UseImage(key, w, h, NRGBA(img))
}

func NRGBA(img *ui.Image) []byte {
	w, h := img.Size()
	pixels := make([]byte, 0, w*h*4)
	for y := range h {
		for x := range w {
			c := img.At(x, y)
			pixels = append(pixels, c.R, c.G, c.B, 255)
		}
	}
	return pixels
}

func Picture(src string, w, h int) gui.View {
	return gui.Image(gui.ImageCfg{Src: src, Width: float32(w), Height: float32(h)})
}
