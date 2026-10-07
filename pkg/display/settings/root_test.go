package settings

import (
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"testing"
)

func TestRootCreatesPagesOnlyWhenOpened(t *testing.T) {
	created := 0
	page := func() shell.View { created++; return &shell.Page{} }
	o := RootOptions{Network: func() string { return "" }, Theme: func() string { return "" }, Assistants: func() string { return "" }, MediaLevel: func() int { return 0 }, Push: func(shell.View) {},
		NetworkPage: page, FeaturesPage: page, DisplayPage: page, HomePage: page, VolumePage: page, AssistantsPage: page, SystemPage: page, DebugPage: page, AboutPage: page,
	}
	rows, actions := RootPage(o).Build()
	if created != 0 || len(rows) != len(actions) {
		t.Fatalf("created=%d rows=%d actions=%d", created, len(rows), len(actions))
	}
	actions[0](0)
	if created != 1 {
		t.Fatalf("created=%d after opening one page", created)
	}
}
