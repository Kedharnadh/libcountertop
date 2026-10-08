package mediacard

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (c *Card) Rail() drawer.Entry {
	return drawer.Entry{
		Name:  func() string { return say.T("rail.media") },
		Order: drawer.OrderPlayer,
		Glyph: func() string { return gogui.IconMusic },
		Open: func() {
			if now := c.options.Player.Now(); now.Playing || now.Paused {
				c.Open()
				return
			}
			c.options.Shell.Push(c.page)
		},
	}
}
