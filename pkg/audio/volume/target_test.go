package volume

import (
	"slices"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type screen schema.Stream

func (s screen) Shows(stream schema.Stream) bool { return schema.Stream(s) == stream }

func sounding(streams ...schema.Stream) func(schema.Stream) bool {
	return func(s schema.Stream) bool { return slices.Contains(streams, s) }
}

func TestTheButtonsAim(t *testing.T) {
	picked := func(s schema.Stream) func() (schema.Stream, bool) {
		return func() (schema.Stream, bool) { return s, true }
	}
	on := func(v any) func() any { return func() any { return v } }

	for _, c := range []struct {
		name string
		aim  Aim
		want schema.Stream
	}{
		{"static ignores what plays", Aim{Mode: schema.ButtonsStatic, Heard: sounding(schema.StreamMedia)}, schema.StreamMain},
		{"static ignores the player card", Aim{Mode: schema.ButtonsStatic, Screen: on(screen(schema.StreamMedia))}, schema.StreamMain},
		{"a picked slider wins in static", Aim{Mode: schema.ButtonsStatic, Picked: picked(schema.StreamAlerts)}, schema.StreamAlerts},
		{"a picked slider wins in dynamic", Aim{Mode: schema.ButtonsDynamic, Picked: picked(schema.StreamAlerts), Heard: sounding(schema.StreamMedia)}, schema.StreamAlerts},
		{"the player card claims media while paused", Aim{Mode: schema.ButtonsDynamic, Screen: on(screen(schema.StreamMedia))}, schema.StreamMedia},
		{"the call screen claims voice", Aim{Mode: schema.ButtonsDynamic, Screen: on(screen(schema.StreamVoice)), Heard: sounding(schema.StreamMedia)}, schema.StreamVoice},
		{"a screen that claims nothing", Aim{Mode: schema.ButtonsDynamic, Screen: on(struct{}{}), Heard: sounding(schema.StreamMedia)}, schema.StreamMedia},
		{"a ringing alarm over music", Aim{Mode: schema.ButtonsDynamic, Heard: sounding(schema.StreamMedia, schema.StreamAlerts)}, schema.StreamAlerts},
		{"a reply over music", Aim{Mode: schema.ButtonsDynamic, Heard: sounding(schema.StreamMedia, schema.StreamVoice)}, schema.StreamVoice},
		{"nothing playing", Aim{Mode: schema.ButtonsDynamic, Heard: sounding()}, schema.StreamMain},
		{"an unset mode is dynamic", Aim{Heard: sounding(schema.StreamMedia)}, schema.StreamMedia},
	} {
		if got := Target(c.aim); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}
