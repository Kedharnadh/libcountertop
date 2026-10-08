package timezone

import (
	"errors"
	"testing"

	tz "github.com/ygelfand/libcountertop/pkg/timezone"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestOverrideSurvivesServerChangesAndCanFollowAgain(t *testing.T) {
	previous := tz.Current()
	t.Cleanup(func() { _ = tz.Use(previous) })
	saved := schema.Time{Home: "UTC0"}
	p := NewPolicy(
		PolicyOptions{
			Read:   func() schema.Time { return saved },
			Home:   func(s string) error { saved.Home = s; return nil },
			Chosen: func(s string) error { saved.Chosen = s; return nil },
		},
	)
	p.SetZone("America/New_York")
	p.FromHome("CET-1")
	want, _ := tz.Named("America/New_York")
	if p.Zone() != "America/New_York" || tz.Current() != want.Spec {
		t.Fatal("server replaced the chosen zone")
	}
	p.SetZone("")
	if tz.Current() != "CET-1" {
		t.Fatalf("following server uses %q", tz.Current())
	}
}

func TestFailedOverrideSavePublishesTheSavedChoice(t *testing.T) {
	previous := tz.Current()
	t.Cleanup(func() { _ = tz.Use(previous) })
	saved := schema.Time{Home: "UTC0"}
	p := NewPolicy(
		PolicyOptions{
			Read:   func() schema.Time { return saved },
			Chosen: func(string) error { return errors.New("write failed") },
		},
	)
	p.SetZone("America/New_York")
	if p.Zone() != "" || p.Select.Get() != FollowHome || tz.Current() != "UTC0" {
		t.Fatal("unsaved timezone became active")
	}
}
