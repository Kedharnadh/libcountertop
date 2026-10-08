package settings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
)

func TestTheExplainingRowDoesNothing(t *testing.T) {
	done := 0
	rows, acts := ConfirmPage("Restart", "what happens", func() { done++ }).Build()

	if len(rows) != 2 {
		t.Fatalf("%d rows on a confirmation, want the explanation and the action", len(rows))
	}
	if acts[0] != nil {
		acts[0](0)
	}
	if done != 0 {
		t.Error("the row explaining what will happen did it instead")
	}
}

func TestTheActionRowActs(t *testing.T) {
	done := 0
	_, acts := ConfirmPage("Restart", "what happens", func() { done++ }).Build()

	acts[1](0)
	if done != 1 {
		t.Errorf("the action ran %d times, want once", done)
	}
}

func TestEveryPowerRowHasAnAction(t *testing.T) {
	rows, acts := PowerPage(func(shell.View) {}, func() {}, func() {}).Build()

	if len(rows) != len(acts) {
		t.Fatalf("%d rows and %d actions", len(rows), len(acts))
	}
	for i, act := range acts {
		if act == nil {
			t.Errorf("row %d (%q) does nothing", i, rows[i].Label)
		}
	}
}
