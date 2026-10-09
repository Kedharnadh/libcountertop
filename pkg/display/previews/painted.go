package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

type painted struct {
	kit  Kit
	key  string
	draw widgets.Paint
}

func (k Kit) Painted(key string, draw widgets.Paint) menu.Preview { return painted{k, key, draw} }

func (p painted) View(_ *gogui.Window, width, height float32) gogui.View {
	pal := p.kit.ui.Palette()
	src := widgets.Painted(p.key, int(width), int(height), pal.Surface, pal, p.draw)
	return widgets.Picture(src, int(width), int(height))
}
