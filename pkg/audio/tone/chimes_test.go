package tone

import (
	"testing"

	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestEveryOfferedChimeHasNotes(t *testing.T) {
	for _, c := range config.Chimes() {
		notes := Wake(c)

		if c == config.ChimeNone {
			if len(notes) != 0 {
				t.Errorf("None sounds %d notes", len(notes))
			}
			continue
		}
		if len(notes) == 0 {
			t.Errorf("%v has no notes", c)
		}
		if Length(notes) == 0 {
			t.Errorf("%v lasts no time", c)
		}
	}
}

func TestTheChimesAreDistinguishable(t *testing.T) {
	seen := map[string]config.Chime{}

	for _, c := range config.Chimes() {
		if c == config.ChimeNone {
			continue
		}

		var key string
		for _, n := range Wake(c) {
			key += string(rune(int(n.Freq))) + string(rune(n.Ms))
		}
		if had, ok := seen[key]; ok {
			t.Errorf("%v sounds exactly like %v", c, had)
		}
		seen[key] = c
	}
}

func TestAnUnknownChimePlaysNothing(t *testing.T) {
	if notes := Wake(config.Chime("harpsichord")); notes != nil {
		t.Errorf("an unknown chime sounds %v", notes)
	}
}
