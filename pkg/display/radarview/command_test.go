package radarview

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/voice"
)

func TestSpokenRadar(t *testing.T) {
	for said, want := range map[string]bool{
		"Radar.":                      true,
		"Open radar":                  true,
		"Show the radar.":             true,
		"Show me the weather radar":   true,
		"the radar is down again":     false,
		"radar":                       true,
		"what does the radar show":    false,
		"open the radar for tomorrow": false,
	} {
		if got := spoken.MatchString(voice.Normalize(said)); got != want {
			t.Errorf("%q matched %v, want %v", said, got, want)
		}
	}
}
