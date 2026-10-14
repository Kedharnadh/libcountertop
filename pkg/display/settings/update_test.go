package settings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/system/firmware"
)

type fakeUpdates struct {
	offer            firmware.Offer
	chose            string
	checks, installs int
}

func (f *fakeUpdates) Version() string       { return "1.0" }
func (f *fakeUpdates) Channels() []string    { return []string{"stable", "dev"} }
func (f *fakeUpdates) Channel() string       { return "dev" }
func (f *fakeUpdates) Choose(c string)       { f.chose = c }
func (f *fakeUpdates) CheckLater()           { f.checks++ }
func (f *fakeUpdates) InstallLater()         { f.installs++ }
func (f *fakeUpdates) Offer() firmware.Offer { return f.offer }

func TestUpdateOffersAnInstallOnlyWhenOneIsAvailable(t *testing.T) {
	for _, c := range []struct {
		finding firmware.Finding
		rows    int
	}{
		{firmware.Unchecked, 3},
		{firmware.Checking, 3},
		{firmware.Current, 3},
		{firmware.Unreachable, 3},
		{firmware.Available, 4},
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
