package previews

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type clockFace struct {
	kit  Kit
	face schema.Face
}

func (k Kit) ClockFace(f schema.Face) menu.Preview { return clockFace{k, f} }

func (c clockFace) View(w *gogui.Window, width, height float32) gogui.View {
	pal := c.kit.ui.Palette()
	tile := gogui.ContainerCfg{
		Width:   width,
		Height:  height,
		Sizing:  gogui.FixedFixed,
		Color:   widgets.Color(pal.Surface),
		Radius:  gogui.RadiusMedium,
		Padding: gogui.NoPadding,
		Clip:    true,
	}
	build, native := clockview.Of(c.face)
	if !native {
		return gogui.Column(tile)
	}
	r := face.Read(time.Now(), c.kit.hours()).Undated()
	tile.Content = []gogui.View{build(w, ui.Rect{W: int(width), H: int(height)}, r, c.kit.ink())}
	return gogui.Column(tile)
}
