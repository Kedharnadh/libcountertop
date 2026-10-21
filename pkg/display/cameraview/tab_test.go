package cameraview

import (
	"context"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type noCameras struct{}

func (noCameras) Cameras(context.Context) ([]homeassistant.Camera, error) { return nil, nil }

type noSite struct{}

func (noSite) URL() string { return "" }

type memory struct{ v schema.Cameras }

func (m *memory) Read() schema.Cameras { return m.v }

func (m *memory) Update(f func(*schema.Cameras)) error {
	f(&m.v)
	return nil
}

func TestTheCamerasTabNeedsPickedCameras(t *testing.T) {
	tabs := dashboard.NewTabs(nil)
	settings := &memory{}
	c := New(Options{
		Shell:    shell.New(),
		Source:   noCameras{},
		Site:     noSite{},
		Settings: settings,
		Tabs:     tabs,
	})
	tabs.Add(20, c.DashboardTabs)

	settings.v.Tab = true
	if got := tabs.List(); len(got) != 0 {
		t.Errorf("tabs %+v with no cameras picked", got)
	}
	settings.v.Pick = schema.HomePick{All: true}
	settings.v.Tab = false
	if got := tabs.List(); len(got) != 0 {
		t.Errorf("tabs %+v with the tab turned off", got)
	}
	settings.v.Tab = true
	got := tabs.List()
	if len(got) != 1 || got[0].Kind != TabKind || got[0].Name == "" {
		t.Fatalf("tabs %+v with the tab on and every camera picked", got)
	}

	tabs.Show(got[0].Key)
	if !c.tabbed() {
		t.Error("the open cameras tab does not count as showing the grid")
	}
	c.o.Shell.Push(c.live)
	if c.tabbed() {
		t.Error("the grid still counts as showing under the live view")
	}
}
