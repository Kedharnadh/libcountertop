package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func pngOf(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestALogoPicksTheArtworkForItsBackground(t *testing.T) {
	light := pngOf(t, 4, 2, color.Black)
	dark := pngOf(t, 6, 2, color.White)
	l := NewLogo(light, dark)

	if got := l.On(theme.RGB(0xffffff)).Bounds().Dx(); got != 4 {
		t.Errorf("on white: %d wide, want the light artwork's 4", got)
	}
	if got := l.On(theme.RGB(0x000000)).Bounds().Dx(); got != 6 {
		t.Errorf("on black: %d wide, want the dark artwork's 6", got)
	}
	if !bytes.Equal(l.PNG(), light) {
		t.Error("PNG is not the light artwork")
	}
}

func TestBrokenArtworkDrawsNothing(t *testing.T) {
	l := NewLogo([]byte("nope"), nil)
	if l.On(theme.RGB(0xffffff)) != nil {
		t.Error("decoded garbage")
	}
}
