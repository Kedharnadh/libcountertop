package tz

import (
	"testing"
	"time"
)

func TestEveryZoneMatchesTheRealOne(t *testing.T) {
	for _, z := range Zones() {
		real, err := time.LoadLocation(z.Name)
		if err != nil {
			t.Skipf("no zone files on this machine: %v", err)
		}

		ours, err := Location(z.Spec)
		if err != nil {
			t.Errorf("%s: %q does not parse: %v", z.Name, z.Spec, err)
			continue
		}

		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		bad := 0

		for at := from; at.Before(from.AddDate(1, 0, 0)); at = at.Add(time.Hour) {
			_, want := at.In(real).Zone()
			_, got := at.In(ours).Zone()

			if got == want {
				continue
			}
			bad++
			if bad == 1 {
				t.Errorf("%s: at %s the offset is %d, want %d (%q)",
					z.Name, at.Format(time.RFC3339), got, want, z.Spec)
			}
		}
		if bad > 1 {
			t.Errorf("%s: %d hours of the year disagree", z.Name, bad)
		}
	}
}

func TestEveryZoneParses(t *testing.T) {
	for _, z := range Zones() {
		if _, err := Location(z.Spec); err != nil {
			t.Errorf("%s: %v", z.Name, err)
		}
	}
}

func TestTheNamesAreDistinct(t *testing.T) {
	seen := map[string]bool{}

	for _, z := range Zones() {
		if seen[z.Name] {
			t.Errorf("%s is offered twice", z.Name)
		}
		seen[z.Name] = true
	}
}

func TestNamedFindsWhatIsOffered(t *testing.T) {
	for _, want := range Zones() {
		got, ok := Named(want.Name)
		if !ok {
			t.Errorf("%s is offered but cannot be looked up", want.Name)
			continue
		}
		if got.Spec != want.Spec {
			t.Errorf("%s resolved to %q, want %q", want.Name, got.Spec, want.Spec)
		}
	}

	if _, ok := Named("Nowhere/Special"); ok {
		t.Error("a zone that is not offered was found")
	}
}

func TestNoRegionOverflowsAPage(t *testing.T) {
	const fits = 13

	for _, r := range Regions() {
		if n := len(In(r)); n > fits {
			t.Errorf("%s has %d zones, more than the %d a page shows", r, n, fits)
		}
	}

	if n := 1 + len(In("")) + len(Regions()); n > fits {
		t.Errorf("the first page has %d rows, more than the %d it shows", n, fits)
	}
}

func TestEveryZoneIsReachable(t *testing.T) {
	seen := map[string]bool{}

	for _, r := range append([]string{""}, Regions()...) {
		for _, z := range In(r) {
			if seen[z.Name] {
				t.Errorf("%s is offered in more than one region", z.Name)
			}
			seen[z.Name] = true
		}
	}

	for _, z := range Zones() {
		if !seen[z.Name] {
			t.Errorf("%s is in no region and not on the first page", z.Name)
		}
	}
}
