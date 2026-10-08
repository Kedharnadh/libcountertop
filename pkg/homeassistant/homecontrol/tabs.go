package homecontrol

import "github.com/ygelfand/libcountertop/pkg/display/dashboard"

func (e *Engine) DashboardTabs() []dashboard.Tab {
	var out []dashboard.Tab
	for _, tab := range e.Dash().Tabs() {
		out = append(out, dashboard.Tab{Kind: TabKind, Key: tab.Key, Name: tab.Name()})
	}
	return out
}
