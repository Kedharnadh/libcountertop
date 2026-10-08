package widgets

import (
	"strings"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

type swatch struct{ light, dark uint32 }

var swatches = map[string]swatch{
	"red":    {0xFADADD, 0x4A2422},
	"orange": {0xFFE1C6, 0x472F17},
	"yellow": {0xFFF0C2, 0x3E3516},
	"green":  {0xD9EEDC, 0x243A26},
	"mint":   {0xD5F0EA, 0x163631},
	"blue":   {0xD9EAFB, 0x1C3448},
	"purple": {0xE9DDF5, 0x342C40},
	"pink":   {0xF7DDEC, 0x46202E},
	"gray":   {0xE5E7EB, 0x2A2D32},
}

func ground(name string, pal theme.Theme) *gogui.Color {
	if name == "none" {
		return &gogui.Color{}
	}
	s, ok := swatches[name]
	if !ok {
		return nil
	}
	rgb := s.light
	if pal.Dark {
		rgb = s.dark
	}
	c := gogui.RGB(uint8(rgb>>16), uint8(rgb>>8), uint8(rgb))
	return &c
}

func secondLine(t tessera.Tile, auto string, h24 bool, now time.Time) string {
	switch {
	case t.Sub == "" || t.Sub == "auto":
		return auto
	case t.Sub == "none":
		return ""
	case strings.HasPrefix(t.Sub, "text:"):
		return strings.TrimPrefix(t.Sub, "text:")
	case strings.HasPrefix(t.Sub, "attr:"):
		if at, ok := t.ExtraNumber("sm"); ok && at > 0 {
			return lastRun(at, now, h24, true)
		}
		return t.ExtraText("s")
	}
	return auto
}
