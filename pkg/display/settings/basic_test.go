package settings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/visual"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestEveryFrameCapSurvivesTheSlider(t *testing.T) {
	for _, fps := range config.MaxFPSSteps {
		if got := config.MaxFPSSteps[FpsIndex(FpsSnap(FpsLevel(fps)))]; got != fps {
			t.Errorf("%d came back as %d", fps, got)
		}
	}
	if FpsIndex(0) != 0 || FpsIndex(100) != len(config.MaxFPSSteps)-1 {
		t.Error("the ends of the track are not 10 and 60")
	}
}

func TestTheSeedSliderEnds(t *testing.T) {
	if SeedOf(0) != 0 || SeedOf(100) != visual.SeedMost || SeedOf(1) < 1 {
		t.Errorf("ends %d %d %d", SeedOf(0), SeedOf(1), SeedOf(100))
	}
	for _, s := range []int{0, 1, 512, visual.SeedMost} {
		if l := SeedLevel(s); (s == 0) != (l == 0) {
			t.Errorf("seed %d sits at %d", s, l)
		}
	}
}
