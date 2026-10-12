package idleview

import (
	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (i *Idle) Rail() drawer.Entry {
	return drawer.Entry{
		Name:  func() string { return say.T("rail.sleep") },
		Order: drawer.OrderIdle,
		Icon:  "mdi:power-sleep",
		Open:  i.Lock,
	}
}
