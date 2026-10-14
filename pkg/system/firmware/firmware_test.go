package firmware

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type manifest struct{ version string }

type source struct {
	offered   string
	fetched   error
	installed error
	restarted string
}

func (s *source) Fetch(context.Context, string) (manifest, error) {
	return manifest{version: s.offered}, s.fetched
}

func (s *source) Describe(m manifest) Release { return Release{Version: m.version} }

func (s *source) Install(_ context.Context, _ manifest, progress func(float32)) error {
	progress(0.5)
	return s.installed
}

func (s *source) Restart(why string) { s.restarted = why }

type trialSource struct {
	*source
	from string
}

func (t trialSource) RolledBack() string { return t.from }
func (t trialSource) Commit()            {}

type settings struct{ schema.Update }

func (s *settings) Read() schema.Update        { return s.Update }
func (s *settings) Channel(v string) error     { s.Update.Channel = v; return nil }
func (s *settings) LastVersion(v string) error { s.Update.LastVersion = v; return nil }

func build(src Source[manifest], saved *settings) *Firmware[manifest] {
	return New(Dependencies[manifest]{
		Source:   src,
		Settings: saved,
		Product:  "Thing",
		Version:  "1.0",
		Channels: []string{"stable", "dev"},
	})
}

func TestAnUnknownSavedChannelFallsBackToTheFirst(t *testing.T) {
	u := build(&source{}, &settings{schema.Update{Channel: "nightly"}})
	if got := u.Channel(); got != "stable" {
		t.Errorf("channel = %q, want stable", got)
	}
	if got := u.channel.Get(); got != "stable" {
		t.Errorf("select shows %q, want stable", got)
	}
}

func TestChoosingAnUnknownChannelChangesNothing(t *testing.T) {
	saved := &settings{schema.Update{Channel: "dev"}}
	u := build(&source{}, saved)
	u.Choose("nightly")
	if saved.Update.Channel != "dev" {
		t.Errorf("saved channel = %q, want dev", saved.Update.Channel)
	}
}

func TestAFailedInstallIsReported(t *testing.T) {
	src := &source{offered: "2.0", installed: errors.New("disk full")}
	failed := 0
	u := New(Dependencies[manifest]{
		Source:    src,
		Settings:  &settings{},
		Version:   "1.0",
		Channels:  []string{"stable"},
		OnFailure: func() { failed++ },
	})

	u.Install(context.Background())

	if failed != 1 {
		t.Errorf("failure feedback ran %d times, want 1", failed)
	}
	if u.Upgrade().Active() {
		t.Error("an upgrade is still shown after it failed")
	}
	if got := u.status.Get(); !strings.Contains(got, "disk full") {
		t.Errorf("status = %q", got)
	}
	if src.restarted != "" {
		t.Error("restarted after a failed install")
	}
}

func TestASuccessfulInstallRestarts(t *testing.T) {
	src := &source{offered: "2.0"}
	u := build(src, &settings{})
	u.Install(context.Background())
	if src.restarted != "update to 2.0" {
		t.Errorf("restart reason = %q", src.restarted)
	}
}

func TestNothingNewInstallsNothing(t *testing.T) {
	src := &source{offered: "1.0"}
	u := build(src, &settings{})
	u.Install(context.Background())
	if src.restarted != "" {
		t.Error("reinstalled the running version")
	}
}

func TestRollbackIsAnnouncedOnlyWithATrial(t *testing.T) {
	plain := build(&source{}, &settings{})
	if slices.Contains(plain.events.Types, EventRolledBack) {
		t.Error("a source without trials offers a rollback event")
	}

	saved := &settings{}
	u := build(trialSource{source: &source{}, from: "1.1"}, saved)
	if !slices.Contains(u.events.Types, EventRolledBack) {
		t.Error("a trial source offers no rollback event")
	}
	u.Announce()
	if got := u.status.Get(); got != "rolled back from 1.1" {
		t.Errorf("status = %q", got)
	}
	if saved.Update.LastVersion != "1.0" {
		t.Errorf("last version = %q, want 1.0", saved.Update.LastVersion)
	}
}

func TestAPendingRebootIsShownOnce(t *testing.T) {
	u := build(&source{}, &settings{})
	u.RebootPending(true)
	u.Announce()
	if got := u.status.Get(); got != "installed 1.0, reboot to finish" {
		t.Errorf("status = %q", got)
	}
}

func TestACheckReportsWhatItFound(t *testing.T) {
	for _, c := range []struct {
		name string
		src  *source
		want Finding
	}{
		{"newer", &source{offered: "2.0"}, Available},
		{"older", &source{offered: "0.9"}, Current},
		{"same", &source{offered: "1.0"}, Current},
		{"empty", &source{}, Current},
		{"offline", &source{offered: "2.0", fetched: errors.New("no route")}, Unreachable},
	} {
		t.Run(c.name, func(t *testing.T) {
			u := build(c.src, &settings{})
			var seen []Finding
			u.Offered.Listen(func(o Offer) { seen = append(seen, o.Finding) })
			u.Check(context.Background())
			if got := u.Offer().Finding; got != c.want {
				t.Errorf("finding = %d, want %d", got, c.want)
			}
			if want := []Finding{Checking, c.want}; !slices.Equal(seen, want) {
				t.Errorf("offered %v, want %v", seen, want)
			}
		})
	}
}
