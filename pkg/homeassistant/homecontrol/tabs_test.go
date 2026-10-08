package homecontrol

import (
	"testing"

	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestDashboardTabsAreTheControlledPicks(t *testing.T) {
	b, _ := testBoard(t)
	if tabs := b.engine.DashboardTabs(); len(tabs) != 0 {
		t.Errorf("tabs %v with nothing picked", tabs)
	}
	if err := testHome.Pick("lights", config.HomePick{Entities: []string{"light.k1"}}); err != nil {
		t.Fatal(err)
	}
	if err := testHome.Control("lights", true); err != nil {
		t.Fatal(err)
	}
	tabs := b.engine.DashboardTabs()
	if len(tabs) != 1 || tabs[0].Kind != TabKind || tabs[0].Key != "lights" || tabs[0].Name == "" {
		t.Errorf("tabs %+v", tabs)
	}
}
