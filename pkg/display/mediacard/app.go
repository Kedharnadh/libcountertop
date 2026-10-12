package mediacard

import (
	"github.com/ygelfand/libcountertop/pkg/display/apps"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (c *Card) App() apps.App {
	return apps.App{
		ID:   schema.AppPlayer,
		Name: func() string { return say.T("rail.media") },
		Icon: "mdi:play-circle",
		Open: func() {
			if now := c.options.Player.Now(); now.Playing || now.Paused {
				c.Open()
				return
			}
			c.options.Shell.Push(c.page)
		},
	}
}
