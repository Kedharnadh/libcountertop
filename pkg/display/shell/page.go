package shell

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Page struct {
	Title string

	Build func() ([]menu.Row, []func(level int))

	Tiles func() ([]menu.Cell, []func(int))

	Preview func(at ui.Rect) bool

	Actions []Action
}

type Action struct {
	Label  string
	Glyph  string
	Danger bool
	Do     func()
}

func (p *Page) Covers() bool { return true }

func (p *Page) Timeout() time.Duration { return SettingsTimeout }
