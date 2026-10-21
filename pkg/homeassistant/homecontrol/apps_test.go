package homecontrol

import (
	"testing"

	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestLightsAndSwitchesAreApps(t *testing.T) {
	e := testEngine()
	got := e.Apps()
	if len(got) != 2 || got[0].ID != config.AppLights || got[1].ID != config.AppSwitches {
		t.Fatalf("apps %+v, want lights then switches", got)
	}
	for _, a := range got {
		if a.Label() == "" || a.Icon == "" {
			t.Errorf("app %q has no name or icon", a.ID)
		}
	}

	got[1].Open()
	full, ok := e.options.Shell.Top().(*Full)
	if !ok || full.Sel.Key != "switches" {
		t.Errorf("opening Switches showed %#v", e.options.Shell.Top())
	}
}
