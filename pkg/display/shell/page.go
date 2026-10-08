package shell

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Page struct {
	Title string

	Build func() ([]model.Row, []func(level int))

	Tiles func() ([]model.Cell, []func(int))

	Preview func(at ui.Rect) bool
}

func (p *Page) Covers() bool { return true }

func (p *Page) Timeout() time.Duration { return SettingsTimeout }
