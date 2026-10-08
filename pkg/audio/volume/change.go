package volume

import (
	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const Step = 5

func Clamp(level int) int { return min(max(level, 0), 100) }

type Change struct {
	Stream schema.Stream
	Level  int
}

type Changes struct{ hook.Hook[Change] }

func (c *Changes) Watch(stream schema.Stream, changed func(int)) func() {
	return c.Listen(func(ch Change) {
		if ch.Stream == stream {
			changed(ch.Level)
		}
	})
}
