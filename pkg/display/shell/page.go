package shell

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Page struct {
	Title string

	Build func() ([]menu.Row, []func(level int))

	Tiles func() ([]menu.Cell, []func(int))

	Preview func(at ui.Rect) Feed

	Side func(*gogui.Window) gogui.View

	Actions []Action
}

type Feed struct {
	Live  bool
	Glyph string
	Note  string
}

type Action struct {
	Label  string
	Glyph  string
	Danger bool
	Do     func()
}

func (p *Page) Covers() bool { return true }

func (p *Page) Timeout() time.Duration { return SettingsTimeout }
