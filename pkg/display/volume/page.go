package volume

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Levels interface {
	Level(schema.Stream) int
	Set(schema.Stream, int)
}
type Chimes interface {
	Chime() schema.Chime
	SetChime(schema.Chime)
	Preview()
}
type Page struct{ *shell.Page }

func (*Page) Shows(schema.Stream) bool { return true }
func NewPage(levels Levels, streams []schema.Stream, chimes Chimes, stack *shell.Shell) *Page {
	streams = slices.Clone(streams)
	return &Page{
		&shell.Page{Title: say.T("settings.volume"), Build: func() ([]model.Row, []func(int)) {
			var rows []model.Row
			var actions []func(int)
			for _, s := range streams {
				rows = append(
					rows,
					model.Row{Label: s.Label(), Kind: model.Slider, Level: levels.Level(s)},
				)
				actions = append(actions, func(level int) { levels.Set(s, level) })
			}
			if chimes != nil {
				rows = append(
					rows,
					model.Row{
						Label: say.T("volume.chime"),
						Kind:  model.Chevron,
						Value: chimes.Chime().Label(),
					},
				)
				actions = append(actions, func(int) { stack.Push(chimePage(chimes)) })
			}
			return rows, actions
		}},
	}
}
func chimePage(chimes Chimes) *shell.Page {
	return &shell.Page{Title: say.T("volume.chime"), Build: func() ([]model.Row, []func(int)) {
		var rows []model.Row
		var actions []func(int)
		for _, c := range schema.Chimes() {
			rows = append(rows, model.Row{Label: c.Label(), Chosen: c == chimes.Chime()})
			actions = append(actions, func(int) { chimes.SetChime(c); chimes.Preview() })
		}
		return rows, actions
	}}
}
