package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

type style string

func (Kit) Style(name string) menu.Preview { return style(name) }

func (s style) View(_ *gogui.Window, width, height float32) gogui.View {
	return widgets.StyleSample(string(s), width, height)
}
