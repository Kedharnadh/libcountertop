package preferences

import (
	"testing"
)

func TestProviderReadsLiveSettingsAndSavesThroughProduct(t *testing.T) {
	live := Settings{YouTube: YouTube{LiveDelay: 12}, Prime: Prime{Persist: true}}
	var saved []string
	p := Provider{
		Read:   func() Settings { return live },
		Prefix: "speaker",
		Save:   func(d, m, v string) error { saved = []string{d, m, v}; return nil },
	}
	if p.Settings().YouTube.LiveDelay != 12 || !p.Settings().Prime.Persist {
		t.Fatal(p.Settings())
	}
	live.YouTube.LiveDelay = 20
	if p.Settings().YouTube.LiveDelay != 20 {
		t.Fatal("cached preferences")
	}
	if err := p.SaveScreens("d", "m", "v"); err != nil || len(saved) != 3 || saved[2] != "v" {
		t.Fatal(saved, err)
	}
	if p.RequestIDPrefix() != "speaker" {
		t.Fatal(p.RequestIDPrefix())
	}
}

func TestAnEmptyProviderFallsBackToDefaults(t *testing.T) {
	var p Provider
	if p.Settings().YouTube.LiveDelay != 10 || p.RequestIDPrefix() != "countertop" {
		t.Fatal(p.Settings(), p.RequestIDPrefix())
	}
	if err := p.SaveScreens("d", "m", "v"); err != nil {
		t.Fatal(err)
	}
}
