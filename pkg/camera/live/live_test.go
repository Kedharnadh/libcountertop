package live

import (
	"strings"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

var levelled = Vendor{
	EVMin:    -1,
	EVMax:    1,
	Scenes:   []string{"auto", "night"},
	Balances: []string{"auto", "daylight"},
	Effects:  []string{"none", "mono"},
	ISOs:     []string{"auto", "100"},
	Levels:   true,
	Rates:    []int{15, 20, 30},
}

var sliding = Vendor{
	EVMin:    -12,
	EVMax:    12,
	Scenes:   []string{"auto", "hdr"},
	Balances: []string{"auto", "shade"},
	Effects:  []string{"none", "aqua"},
	Sliders:  true,
	Noises:   []string{"off", "fast", "high_quality"},
	Rates:    []int{15, 24, 30},
}

type store struct{ cam schema.Camera }

func (s *store) camera(o Options) *Camera {
	s.cam = schema.DefaultCamera()
	o.Saved = func() schema.Camera { return s.cam }
	return New(o)
}

func TestALevelledHALIsToldLevelsAndNoSliders(t *testing.T) {
	var s store
	c := s.camera(
		Options{
			Vendor:  levelled,
			Streams: Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
		},
	)
	p := c.Params(c.Saved())
	for _, want := range []string{"iso-speed=auto", "brightness=middle", "edge=middle", "fps-range=auto"} {
		if !strings.Contains(p, want) {
			t.Errorf("%q is missing %s", p, want)
		}
	}
	if strings.Contains(p, "noise-reduction") || strings.Contains(p, "sharpness=") {
		t.Errorf("%q carries slider knobs", p)
	}
}

func TestASlidingHALIsToldSlidersAndNoLevels(t *testing.T) {
	var s store
	c := s.camera(
		Options{
			Vendor:  sliding,
			Streams: Streams{Main: []string{"1600x1200"}, MainDefault: "1600x1200"},
		},
	)
	p := c.Params(c.Saved())
	for _, want := range []string{"brightness=5", "sharpness=12", "noise-reduction=1"} {
		if !strings.Contains(p, want) {
			t.Errorf("%q is missing %s", p, want)
		}
	}
	if strings.Contains(p, "iso-speed") || strings.Contains(p, "edge=") {
		t.Errorf("%q carries level knobs", p)
	}
	if _, ok := c.Table().Find("iso"); ok {
		t.Error("an iso knob on a HAL with no ISO choices")
	}
}

func TestWithoutSubWidthsThereIsNoSubStream(t *testing.T) {
	var s store
	c := s.camera(
		Options{
			Vendor:  levelled,
			Streams: Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
		},
	)
	for _, name := range []string{"sub_on", "sub_size"} {
		if _, ok := c.Table().Find(name); ok {
			t.Errorf("%s is offered with no sub stream", name)
		}
	}
	if got := c.Served(); len(got) != 1 || got[0] != 0 {
		t.Errorf("serves %v, want only the main stream", got)
	}
	if snap := c.snapshot(); snap.Config.SubWidth != 0 || snap.Config.Turn != noTurn {
		t.Errorf("snapshot %+v", snap.Config)
	}
}

func TestASubStreamFollowsTheMainShape(t *testing.T) {
	var s store
	c := s.camera(Options{
		Vendor: sliding,
		Streams: Streams{
			Main:        []string{"1600x1200"},
			MainDefault: "1600x1200",
			Sub:         []int{640, 320},
			SubDefault:  640,
		},
		Turn: func() int { return 1 },
	})
	if got := c.Served(); len(got) != 2 {
		t.Errorf("serves %v, want both streams", got)
	}
	snap := c.snapshot()
	if snap.Config.Width != 1200 || snap.Config.Height != 1600 {
		t.Errorf("a quarter turn sends %dx%d", snap.Config.Width, snap.Config.Height)
	}
	if snap.Config.SubWidth != 480 || snap.Config.SubHeight != 640 {
		t.Errorf("the sub stream is %dx%d", snap.Config.SubWidth, snap.Config.SubHeight)
	}
}

func TestStillsWaitForAKeyframeWhenTheHelperNeedsOne(t *testing.T) {
	var s store
	o := Options{
		Vendor:  levelled,
		Streams: Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
	}
	if extra := s.camera(o).snapshot().StillExtra; extra != 0 {
		t.Errorf("waits %s more without being asked", extra)
	}
	o.StillOnKeyframe = true
	if extra := s.camera(o).snapshot().StillExtra; extra != Keyframe*time.Second {
		t.Errorf("waits %s more, want a keyframe interval", extra)
	}
}

func TestARateSnapsToWhatTheHALOffers(t *testing.T) {
	var s store
	c := s.camera(
		Options{
			Vendor:  levelled,
			Streams: Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
		},
	)
	k := c.Saved()
	k.RateAuto, k.Rate = false, 24
	if p := c.Params(
		k,
	); !strings.Contains(p, "fps-range=20,20") &&
		!strings.Contains(p, "fps-range=30,30") {
		t.Errorf("24 fps became %q", p)
	}
}
