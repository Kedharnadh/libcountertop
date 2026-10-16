package settings

import (
	"context"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/inference/wake"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type wakeMemory struct{ wake config.Wake }

func (m *wakeMemory) Read() config.Wake { return m.wake }

func (m *wakeMemory) Update(change func(*config.Wake)) error {
	change(&m.wake)
	return nil
}

func thresholds() (*AssistantPages, *wakeMemory) {
	m := &wakeMemory{wake: config.DefaultWake()}
	return NewAssistantPages(AssistantDependencies{
		Slots:       2,
		Preferences: Preferences{Wake: m},
	}), m
}

func TestTheSensitivitySliderIsTheThresholdTurnedAround(t *testing.T) {
	for _, tc := range []struct {
		was  float64
		want int
	}{
		{0.85, 64},
		{0.5, MostSure},
		{0.99, LeastSure},
		{0, MostSure},
		{1, LeastSure},
		{2, LeastSure},
	} {
		if got := Sensitivity(tc.was); got != tc.want {
			t.Errorf("a threshold of %v shows as %d, want %d", tc.was, got, tc.want)
		}
	}
}

func TestDraggingToTheEndsSavesAUsableThreshold(t *testing.T) {
	p, m := thresholds()

	for _, level := range []int{0, 50, 85, 99, 100} {
		p.Threshold(0)(level)

		got := m.Read().Slot(0).Threshold
		if got < 0.5 || got > 0.99 {
			t.Errorf("dragging to %d saved %v, outside what a model can use", level, got)
		}
	}
}

func TestTheSliderShowsWhatItSaved(t *testing.T) {
	p, m := thresholds()

	for _, level := range []int{50, 70, 85, 99} {
		p.Threshold(1)(level)

		if got := Sensitivity(m.Read().Slot(1).Threshold); got != level {
			t.Errorf("dragged to %d, reads back as %d", level, got)
		}
	}
}

func TestEachSlotSetsItsOwnThreshold(t *testing.T) {
	p, m := thresholds()

	p.Threshold(0)(60)
	p.Threshold(1)(90)

	if got := m.Read().Slot(0).Threshold; got != 0.89 {
		t.Errorf("slot 1 is at %v after only slot 2 was changed", got)
	}
	if got := m.Read().Slot(1).Threshold; got != 0.59 {
		t.Errorf("slot 2 is at %v", got)
	}
}

type chooser struct{ allowed bool }

func (c chooser) Allowed() bool                           { return c.allowed }
func (chooser) Choose(context.Context, int, string) error { return nil }

func TestTheWakeWordRowNeedsActions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed bool
	}{
		{"actions off", false},
		{"actions on", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewAssistantPages(AssistantDependencies{
				Slots:       2,
				Preferences: Preferences{Wake: &wakeMemory{wake: config.DefaultWake()}},
				Models:      func() []wake.Model { return nil },
				Phrases:     func() []string { return []string{"Okay Nabu"} },
				Chooser:     chooser{allowed: tc.allowed},
			})
			row, _ := p.wordRow(0)
			if row.Dim == tc.allowed {
				t.Errorf("dim %v with actions allowed %v", row.Dim, tc.allowed)
			}
			if !tc.allowed && row.Hint != say.T("assistants.word.ha") {
				t.Errorf("hint %q, want the enable-actions hint", row.Hint)
			}
		})
	}
}
