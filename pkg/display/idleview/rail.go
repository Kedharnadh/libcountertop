package idleview

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (i *Idle) Rail() drawer.Entry {
	return drawer.Entry{
		Name:  func() string { return say.T("rail.sleep") },
		Order: drawer.OrderIdle,
		Glyph: func() string { return gogui.IconMoon },
		Open:  i.Lock,
	}
}
