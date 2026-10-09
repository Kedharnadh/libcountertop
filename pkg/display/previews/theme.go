package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

type palette theme.Theme

func (Kit) Theme(t theme.Theme) menu.Preview { return palette(t) }

func (p palette) View(_ *gogui.Window, width, height float32) gogui.View {
	return widgets.Swatch(theme.Theme(p), width, height)
}
