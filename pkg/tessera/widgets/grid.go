package widgets

import (
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type cell struct {
	page          int
	column, row   int
	columns, rows int
}

func place(slot int, size string, g schema.TesseraGrid) cell {
	cells := g.Columns * g.Rows
	c := cell{page: slot / cells, columns: 1, rows: 1}
	at := slot % cells
	c.column, c.row = at%g.Columns, at/g.Columns
	switch size {
	case "full":
		c.column, c.row, c.columns, c.rows = 0, 0, g.Columns, g.Rows
	case "wide":
		c.columns = min(2, g.Columns)
	case "tall":
		c.rows = 2
	case "square":
		c.columns, c.rows = 2, 2
	default:
		if len(size) == 3 && size[1] == 'x' {
			c.columns, c.rows = int(size[0]-'0'), int(size[2]-'0')
		}
	}
	c.columns = max(1, min(c.columns, g.Columns-c.column))
	c.rows = max(1, min(c.rows, g.Rows-c.row))
	return c
}

func (c cell) rect(in ui.Rect, g schema.TesseraGrid, gap int) ui.Rect {
	w := (in.W - gap*(g.Columns-1)) / g.Columns
	h := (in.H - gap*(g.Rows-1)) / g.Rows
	return ui.Rect{
		X: in.X + c.column*(w+gap),
		Y: in.Y + c.row*(h+gap),
		W: c.columns*w + (c.columns-1)*gap,
		H: c.rows*h + (c.rows-1)*gap,
	}
}
