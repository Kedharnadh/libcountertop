package ui

import (
	"fmt"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func BenchmarkDrawText(b *testing.B) {
	f := MustLoad(Regular, 28)
	dst := NewImage(480, 64, theme.Color{})

	fg := theme.Color{R: 0xe6, G: 0xec, B: 0xf5}
	bg := theme.Color{R: 0x1e, G: 0x22, B: 0x2c}

	const label = "Speaker volume"

	b.Run("cold", func(b *testing.B) {
		for b.Loop() {
			f.mu.Lock()
			clear(f.glyphs)
			f.mu.Unlock()

			DrawText(dst, f, 0, 0, fg, bg, label)
		}
	})

	b.Run("cached", func(b *testing.B) {
		for b.Loop() {
			DrawText(dst, f, 0, 0, fg, bg, label)
		}
	})

	b.Run("changing", func(b *testing.B) {
		n := 0
		for b.Loop() {
			n = (n + 1) % 101
			DrawText(dst, f, 0, 0, fg, bg, fmt.Sprintf("Volume %d%%", n))
		}
	})

	b.Run("clipped away", func(b *testing.B) {
		away := &clipped{Image: dst, to: Rect{X: 0, Y: 0, W: 1, H: 1}}
		for b.Loop() {
			DrawText(away, f, 200, 200, fg, bg, label)
		}
	})
}

type clipped struct {
	*Image
	to Rect
}

func (c *clipped) Clipped() Rect { return c.to }

func TestGlyphCacheKeepsItsOwnCoverage(t *testing.T) {
	f := MustLoad(Regular, 24)

	first := f.glyph('W')
	before := append([]byte(nil), first.cover.Pix...)

	for _, r := range "ilMq8#" {
		f.glyph(r)
	}

	again := f.glyph('W')
	if string(again.cover.Pix) != string(before) {
		t.Fatal("a cached glyph changed after other runes were rasterized")
	}
}

func TestMeasureMatchesWhereGlyphsLand(t *testing.T) {
	f := MustLoad(Regular, 28)

	for _, s := range []string{"", " ", "8", "Volume 100%", "Wavy AVA To."} {
		w, h := f.Measure(s)

		if want := f.width(s).Round(); w != want {
			t.Errorf("Measure(%q) width = %d, want %d", s, w, want)
		}
		if h != f.ascent+f.descent {
			t.Errorf("Measure(%q) height = %d, want %d", s, h, f.ascent+f.descent)
		}
	}
}
