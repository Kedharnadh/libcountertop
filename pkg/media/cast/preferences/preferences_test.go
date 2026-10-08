package preferences

import (
	"testing"
)

func TestProviderReadsLiveSettingsAndSavesThroughProduct(t *testing.T) {
	old := snapshot()
	defer Configure(old)
	live := Settings{YouTube: YouTube{LiveDelay: 12}, Prime: Prime{Persist: true}}
	var saved []string
	Configure(
		Provider{
			Read:            func() Settings { return live },
			RequestIDPrefix: "speaker",
			SaveScreens:     func(d, m, v string) error { saved = []string{d, m, v}; return nil },
		},
	)
	if Get().YouTube.LiveDelay != 12 || !Get().Prime.Persist {
		t.Fatal(Get())
	}
	live.YouTube.LiveDelay = 20
	if Get().YouTube.LiveDelay != 20 {
		t.Fatal("cached preferences")
	}
	if err := SaveScreens("d", "m", "v"); err != nil || len(saved) != 3 || saved[2] != "v" {
		t.Fatal(saved, err)
	}
	if RequestIDPrefix() != "speaker" {
		t.Fatal(RequestIDPrefix())
	}
}
