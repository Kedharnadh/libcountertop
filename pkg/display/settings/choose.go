package settings

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func Choose[T interface {
	comparable
	Label() string
}](title string, values []T, read func() T, save func(T), preview func(T) func(ui.Surface, ui.Rect, theme.Theme)) *shell.Page {
	values = slices.Clone(values)
	return &shell.Page{Title: title, Build: func() ([]model.Row, []func(int)) {
		selected := read()
		rows := make([]model.Row, 0, len(values))
		actions := make([]func(int), 0, len(values))
		for _, value := range values {
			row := model.Row{Label: value.Label(), Chosen: value == selected}
			if preview != nil {
				row.Preview = preview(value)
			}
			rows = append(rows, row)
			actions = append(actions, func(int) { save(value) })
		}
		return rows, actions
	}}
}
