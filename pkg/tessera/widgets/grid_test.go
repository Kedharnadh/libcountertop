package widgets

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func TestSlotsLandOnTheirPageAndCell(t *testing.T) {
	g := schema.TesseraGrid{Columns: 5, Rows: 8}
	tests := []struct {
		slot int
		size string
		want cell
	}{
		{0, "single", cell{page: 0, columns: 1, rows: 1}},
		{7, "wide", cell{page: 0, column: 2, row: 1, columns: 2, rows: 1}},
		{40, "single", cell{page: 1, columns: 1, rows: 1}},
		{4, "square", cell{page: 0, column: 4, columns: 1, rows: 2}},
		{3, "full", cell{page: 0, columns: 5, rows: 8}},
		{0, "3x2", cell{page: 0, columns: 3, rows: 2}},
	}
	for _, tt := range tests {
		if got := place(tt.slot, tt.size, g); got != tt.want {
			t.Errorf("slot %d %s = %+v, want %+v", tt.slot, tt.size, got, tt.want)
		}
	}
}

func TestCellsTileTheArea(t *testing.T) {
	g := schema.TesseraGrid{Columns: 2, Rows: 2}
	in := ui.Rect{X: 10, Y: 20, W: 210, H: 110}
	got := cell{column: 1, row: 1, columns: 1, rows: 1}.rect(in, g, 10)
	if got != (ui.Rect{X: 120, Y: 80, W: 100, H: 50}) {
		t.Errorf("bottom right cell %+v", got)
	}
}

func TestPageLinksNameTheirPage(t *testing.T) {
	for entity, want := range map[string]int{
		"screen.page_1": 1, "screen.page_12": 12, "screen.page_0": 0, "screen.page_123": 0, "light.page_1": 0,
	} {
		if got := pageNumber(entity); got != want {
			t.Errorf("pageNumber(%q) = %d, want %d", entity, got, want)
		}
	}
}

func TestANewLayoutKeepsThePageOnScreen(t *testing.T) {
	v := New(Dependencies{})
	first := tessera.Layout{Revision: "a", Home: 0, Pages: []tessera.Page{{ID: "p1"}, {ID: "p2"}}}
	v.showing(first)
	v.open(first, 1)
	moved := tessera.Layout{
		Revision: "b",
		Home:     0,
		Pages:    []tessera.Page{{ID: "p0"}, {ID: "p1"}, {ID: "p2"}},
	}
	if got := v.showing(moved); got != 2 {
		t.Errorf("after a resend the page is %d, want 2 (p2)", got)
	}
	gone := tessera.Layout{Revision: "c", Home: 1, Pages: []tessera.Page{{ID: "p0"}, {ID: "p1"}}}
	if got := v.showing(gone); got != 1 {
		t.Errorf("with its page deleted the screen shows %d, want home 1", got)
	}
}

func TestTheScreensaverReturnsHomeOnlyWhenAsked(t *testing.T) {
	back := false
	v := New(Dependencies{HomeAfterIdle: func() bool { return back }})
	l := tessera.Layout{Revision: "a", Home: 0, Pages: []tessera.Page{{ID: "p1"}, {ID: "p2"}}}
	v.showing(l)
	v.open(l, 1)
	v.Rested()
	if got := v.showing(l); got != 1 {
		t.Errorf("with the toggle off the screensaver moved the page to %d", got)
	}
	back = true
	v.Rested()
	if got := v.showing(l); got != 0 {
		t.Errorf("with the toggle on the screensaver left the page at %d", got)
	}
}
