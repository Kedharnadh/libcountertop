package menu

import (
	"image"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Mark struct {
	SVG   string
	Image image.Image
}

type Kind int

const (
	Plain Kind = iota
	Chevron
	Toggle
	Slider
	Field
	Segments
)

type Row struct {
	Glyph string
	Label string
	Hint  string

	Snap func(level int) int

	Value string

	Options []string
	Marks   []Mark

	Kind  Kind
	On    bool
	Level int

	Chosen bool
	Dim    bool

	Preview func(s ui.Surface, at ui.Rect, palette theme.Theme)

	Save func(string)
}

type Preview interface {
	View(w *gogui.Window, width, height float32) gogui.View
}

type Cell struct {
	Label   string
	Chosen  bool
	Preview Preview
	Edit    func()
}
