package apps

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

func named(id string) App {
	return App{ID: id, Name: func() string { return id }, Glyph: func() string { return "" }}
}

func rig(t *testing.T) (*Apps, *storage.Store[schema.Apps], *shell.Shell) {
	t.Helper()
	st, err := storage.Load(filepath.Join(t.TempDir(), "state.json"), schema.DefaultApps(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := shell.New()
	t.Cleanup(s.Close)
	a := New(Options{
		Settings: storage.Bind(
			func() *storage.Store[schema.Apps] { return st },
			func(c *schema.Apps) *schema.Apps { return c },
		),
		Shell: s,
	})
	for _, id := range []string{schema.AppPlayer, schema.AppCalls, "notes"} {
		a.Add(named(id))
	}
	return a, st, s
}

func appIDs(list []App) []string {
	out := make([]string, len(list))
	for n, app := range list {
		out[n] = app.ID
	}
	return out
}

func labels(entries []drawer.Entry) []string {
	out := make([]string, len(entries))
	for n, e := range entries {
		out[n] = e.Label()
	}
	return out
}

func TestTheSavedOrderComesFirstAndUnknownAppsAreSkipped(t *testing.T) {
	got := appIDs(ordered(
		[]App{named("player"), named("calls"), named("notes")},
		[]string{"gone", "notes", "player"},
	))
	if want := []string{"notes", "player", "calls"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

func TestAMoveSavesTheWholeOrder(t *testing.T) {
	a, st, _ := rig(t)
	a.Move(2, 0)
	if got, want := st.Get().Order, []string{"notes", "player", "calls"}; !slices.Equal(got, want) {
		t.Errorf("saved %v, want %v", got, want)
	}
	a.Move(0, 9)
	if got := appIDs(a.List()); got[0] != "notes" {
		t.Errorf("a move past the end changed the order to %v", got)
	}
}

func TestTheDockShowsFavoritesInAppOrder(t *testing.T) {
	a, _, _ := rig(t)
	if got, want := labels(a.Dock()), []string{"player", "calls"}; !slices.Equal(got, want) {
		t.Fatalf("dock %v, want %v by default", got, want)
	}

	a.Move(1, 0)
	a.SetDocked("notes", true)
	a.SetDocked(schema.AppPlayer, false)
	dock := a.Dock()
	if got, want := labels(dock), []string{"calls", "notes"}; !slices.Equal(got, want) {
		t.Errorf("dock %v, want %v", got, want)
	}
	for n, e := range dock {
		if e.Order != drawer.OrderFavorites+n {
			t.Errorf("%s sits at %d, want %d", e.Label(), e.Order, drawer.OrderFavorites+n)
		}
	}
	if !a.Docked("notes") || a.Docked(schema.AppPlayer) {
		t.Error("Docked does not follow the saved dock")
	}
}

func TestTheAppsEntryOpensTheTiles(t *testing.T) {
	a, _, s := rig(t)
	entry := a.Rail()
	if entry.Order != drawer.OrderApps || entry.Glyph() == "" {
		t.Fatalf("entry %+v", entry)
	}
	entry.Open()
	screens := a.Screens(nil)
	if s.Top() != shell.View(a.view) || screens(s.Top()) == nil {
		t.Error("the entry did not open the apps view")
	}
	if screens(&View{}) != nil {
		t.Error("another view got the apps screen")
	}
}

func TestADropLandsOnTheCellUnderTheFinger(t *testing.T) {
	g := grid{cols: 3, side: 100, gap: 10}
	for _, c := range []struct {
		from   int
		dx, dy float32
		want   int
	}{
		{0, 0, 0, 0},
		{0, 110, 0, 1},
		{0, 220, 110, 5},
		{4, -500, -500, 0},
		{0, 500, 0, 2},
		{0, 0, 900, 6},
	} {
		if got := g.drop(c.from, c.dx, c.dy, 7); got != c.want {
			t.Errorf("from %d by (%v,%v) dropped at %d, want %d", c.from, c.dx, c.dy, got, c.want)
		}
	}
}

func TestTheRowsReadAndWriteIDs(t *testing.T) {
	tb := NewTable()
	var c schema.Apps
	if err := tb.MustRow("dock").Write(&c, " calls, ,notes "); err != nil {
		t.Fatal(err)
	}
	if want := []string{"calls", "notes"}; !slices.Equal(c.Dock, want) {
		t.Errorf("dock %v, want %v", c.Dock, want)
	}
	if got := tb.MustRow("dock").Read(&c); got != "calls,notes" {
		t.Errorf("read %q", got)
	}
}
