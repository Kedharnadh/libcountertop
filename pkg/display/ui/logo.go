package ui

import (
	"bytes"
	"image"
	_ "image/png"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

type Logo struct {
	light, dark artwork
}

type artwork struct {
	raw  []byte
	once sync.Once
	img  image.Image
}

func NewLogo(light, dark []byte) *Logo {
	return &Logo{light: artwork{raw: light}, dark: artwork{raw: dark}}
}

func (l *Logo) PNG() []byte { return l.light.raw }

func (l *Logo) Light() image.Image { return l.light.decode() }

func (l *Logo) Dark() image.Image { return l.dark.decode() }

func (l *Logo) On(bg theme.Color) image.Image {
	if theme.Dark(bg) {
		return l.Dark()
	}
	return l.Light()
}

func (l *Logo) Draw(s Surface, r Rect, on theme.Color) { DrawImageFit(s, l.On(on), r, on) }

func (a *artwork) decode() image.Image {
	a.once.Do(func() {
		img, _, err := image.Decode(bytes.NewReader(a.raw))
		if err == nil {
			a.img = img
		}
	})
	return a.img
}
