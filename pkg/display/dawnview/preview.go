package dawnview

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log/slog"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

//go:embed thumbs/*.jpg
var thumbs embed.FS

type Preview schema.WakeLook

func (p Preview) View(_ *gogui.Window, width, height float32) gogui.View {
	src := p.picture(width, height)
	if src == "" {
		return gogui.Column(
			gogui.ContainerCfg{Width: width, Height: height, Sizing: gogui.FixedFixed},
		)
	}
	return gogui.Image(gogui.ImageCfg{Src: src, Width: width, Height: height})
}

func (p Preview) picture(width, height float32) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	key := fmt.Sprintf("wake/%s/%.3f", string(p), width/height)
	if gogui.HasImage(key) {
		return "mem:" + key
	}
	raw, err := thumbs.ReadFile("thumbs/" + string(p) + ".jpg")
	if err != nil {
		slog.Warn("a wake light preview is missing", "look", string(p), "err", err)
		return ""
	}
	full, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		slog.Warn("a wake light preview would not decode", "look", string(p), "err", err)
		return ""
	}
	b := full.Bounds()
	cw, ch := b.Dx(), b.Dy()
	if want := float64(width) / float64(height); float64(cw)/float64(ch) > want {
		cw = int(float64(ch) * want)
	} else {
		ch = int(float64(cw) / want)
	}
	at := image.Pt(b.Min.X+(b.Dx()-cw)/2, b.Min.Y+(b.Dy()-ch)/2)
	img := image.NewNRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(img, img.Bounds(), full, at, draw.Src)
	return gogui.UseImage(key, cw, ch, img.Pix)
}
