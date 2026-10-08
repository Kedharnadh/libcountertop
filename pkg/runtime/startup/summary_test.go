package startup

import "testing"

func TestSummaryChangesWithEachStage(t *testing.T) {
	looking := []Progress{{Name: "wifi", Doing: "looking for the network"}}
	asking := []Progress{{Name: "wifi", Doing: "asking for an address"}}

	if Summary(looking) == Summary(asking) {
		t.Error("two stages of the same component summarize alike")
	}
}

func TestSummaryChangesWhenSomethingFinishes(t *testing.T) {
	waiting := []Progress{{Name: "wifi", Doing: "looking"}}
	done := []Progress{{Name: "wifi", Doing: "looking", Done: true}}

	if Summary(waiting) == Summary(done) {
		t.Error("a component finishing did not change the summary")
	}
}

func TestSummaryIsStableWhileNothingChanges(t *testing.T) {
	first := []Progress{
		{Name: "wifi", Doing: "looking"},
		{Name: "dhcp", Doing: "asking", Done: true},
	}
	second := []Progress{
		{Name: "wifi", Doing: "looking"},
		{Name: "dhcp", Doing: "asking", Done: true},
	}

	if Summary(first) != Summary(second) {
		t.Errorf("the same progress summarized as %q and %q", Summary(first), Summary(second))
	}
}

func TestSummaryOfEverythingDone(t *testing.T) {
	all := []Progress{
		{Name: "wifi", Doing: "looking", Done: true},
		{Name: "dhcp", Doing: "asking", Done: true},
	}

	if got := Summary(all); got == "" {
		t.Errorf("summary = %q, want completed rows retained for redraw tracking", got)
	}
	if Summary(nil) != "" {
		t.Error("nothing to wait for should summarize to nothing")
	}
}

func TestEverythingDoneIsNotTheStartingSummary(t *testing.T) {
	const starts = "\x00"

	if Summary(nil) == starts {
		t.Error("the starting summary is reachable, so the first pass may not draw")
	}
}
