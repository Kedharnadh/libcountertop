package visualinput

import (
	"errors"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/visual"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestFailedSaveRestoresActiveSettings(t *testing.T) {
	active := config.Visual{Kind: string(visual.Orb), Label: "saved", Seed: 3}
	fail := errors.New("write failed")
	v := New(Options{MicRate: 16000, SpeakerRate: 48000, SpeakerChannels: 2,
		Settings: failedSettings{active: active, err: fail}, Lift: new(lift),
	})
	v.Restore(active)
	v.SetLabel("unsaved")
	v.SetKind(visual.Kind("invalid"))
	v.SetSeed(9)
	if v.Kind() != visual.Orb || v.named != "saved" {
		t.Fatalf("unsaved settings became active: kind=%s label=%s", v.Kind(), v.named)
	}
}

type failedSettings struct {
	active config.Visual
	err    error
}

func (s failedSettings) Read() config.Visual               { return s.active }
func (s failedSettings) Update(func(*config.Visual)) error { return s.err }

type lift int

func (l *lift) Read() int { return int(*l) }

func (l *lift) Update(change func(*int)) error {
	n := int(*l)
	change(&n)
	*l = lift(n)
	return nil
}
