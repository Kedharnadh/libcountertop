package visualinput

import (
	"errors"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"testing"
)

func TestFailedSaveRestoresActiveSettings(t *testing.T) {
	active := config.Visual{Kind: string(visual.Orb), Label: "saved", Seed: 3}
	fail := errors.New("write failed")
	v := New(Options{MicRate: 16000, SpeakerRate: 48000, SpeakerChannels: 2,
		Read: func() config.Visual { return active },
		Kind: func(string) error { return fail }, Label: func(string) error { return fail }, Seed: func(int) error { return fail },
	})
	v.Restore(active)
	v.SetLabel("unsaved")
	v.SetKind(visual.Kind("invalid"))
	v.SetSeed(9)
	if v.Kind() != visual.Orb || v.named != "saved" {
		t.Fatalf("unsaved settings became active: kind=%s label=%s", v.Kind(), v.named)
	}
}
