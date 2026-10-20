package settings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/network/releases"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/system/firmware"
)

type fakeUpdates struct {
	offer            firmware.Offer
	catalog          firmware.Catalog
	chose, built     string
	checks, installs int
}

func (f *fakeUpdates) Version() string       { return "1.0" }
func (f *fakeUpdates) Channels() []string    { return []string{"stable", "dev"} }
func (f *fakeUpdates) Channel() string       { return "dev" }
func (f *fakeUpdates) Choose(c string)       { f.chose = c }
func (f *fakeUpdates) CheckLater()           { f.checks++ }
func (f *fakeUpdates) InstallLater()         { f.installs++ }
func (f *fakeUpdates) Offer() firmware.Offer { return f.offer }

func (f *fakeUpdates) Catalog() firmware.Catalog  { return f.catalog }
func (f *fakeUpdates) ListLater()                 {}
func (f *fakeUpdates) InstallBuildLater(v string) { f.built = v }

func TestUpdateOffersAnInstallOnlyWhenOneIsAvailable(t *testing.T) {
	for _, c := range []struct {
		finding firmware.Finding
		rows    int
	}{
		{firmware.Unchecked, 4},
		{firmware.Checking, 4},
		{firmware.Current, 4},
		{firmware.Unreachable, 4},
		{firmware.Available, 5},
	} {
		u := &fakeUpdates{
			offer: firmware.Offer{Finding: c.finding, Release: firmware.Release{Version: "2.0"}},
		}
		rows, acts := NewBasicPages(BasicDependencies{Updates: u}).Update().Build()
		if len(rows) != c.rows || len(acts) != c.rows {
			t.Errorf(
				"finding %d: rows=%d acts=%d, want %d",
				c.finding,
				len(rows),
				len(acts),
				c.rows,
			)
		}
		if rows[1].Level != 1 {
			t.Errorf("finding %d: channel segment at %d, want 1", c.finding, rows[1].Level)
		}
		acts[2](0)
		if want := map[bool]int{true: 0, false: 1}[c.finding == firmware.Checking]; u.checks != want {
			t.Errorf("finding %d: checks=%d, want %d", c.finding, u.checks, want)
		}
		if c.finding == firmware.Available {
			acts[3](0)
			if u.installs != 1 {
				t.Errorf("installs=%d, want 1", u.installs)
			}
		}
		acts[1](0)
		if u.chose != "stable" {
			t.Errorf("finding %d: chose %q, want stable", c.finding, u.chose)
		}
	}
}

func TestVersionsInstallsAnythingButTheRunningOne(t *testing.T) {
	u := &fakeUpdates{
		catalog: firmware.Catalog{Finding: firmware.Current, Builds: []releases.Release{
			{Tag: "1.1-dev.2", Prerelease: true},
			{Tag: "1.0"},
			{Tag: "0.9"},
		}},
	}
	sh := shell.New()
	rows, acts := NewBasicPages(BasicDependencies{Updates: u, Shell: sh}).Versions().Build()
	if len(rows) != 3 || len(acts) != 3 {
		t.Fatalf("rows=%d acts=%d, want 3", len(rows), len(acts))
	}
	if acts[1] != nil {
		t.Error("the running version can be installed")
	}
	acts[2](0)
	if u.built != "" {
		t.Fatalf("installed %q before confirming", u.built)
	}
	confirm, ok := sh.Top().(*shell.Page)
	if !ok {
		t.Fatalf("top is %T, want the confirm page", sh.Top())
	}
	_, confirmActs := confirm.Build()
	confirmActs[len(confirmActs)-1](0)
	if u.built != "0.9" {
		t.Errorf("installed %q, want 0.9", u.built)
	}
}

func TestVersionsSaysWhyItListsNothing(t *testing.T) {
	for _, c := range []struct {
		finding firmware.Finding
		want    string
	}{
		{firmware.Checking, "update.checking"},
		{firmware.Current, "update.versions.none"},
		{firmware.Unreachable, "update.unreachable"},
	} {
		u := &fakeUpdates{catalog: firmware.Catalog{Finding: c.finding}}
		rows, acts := NewBasicPages(BasicDependencies{Updates: u}).Versions().Build()
		if len(rows) != 1 || acts[0] != nil {
			t.Fatalf("finding %d: rows=%d, want one plain row", c.finding, len(rows))
		}
		if got := rows[0].Label; got != say.T(c.want) {
			t.Errorf("finding %d: says %q, want %q", c.finding, got, say.T(c.want))
		}
	}
}
