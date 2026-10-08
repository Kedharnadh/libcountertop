package ui

import (
	"image"
	"testing"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func empty() {
	iconMu.Lock()
	defer iconMu.Unlock()

	iconCache = map[iconKey]*image.Alpha{}
	iconBytes = 0
}

func held() (entries, bytes int) {
	iconMu.Lock()
	defer iconMu.Unlock()

	return len(iconCache), iconBytes
}

func TestTheSameMaskIsNotRasterizedTwice(t *testing.T) {
	empty()

	mask(icons.ActionInfoOutline, 64, 64)
	was, _ := held()

	mask(icons.ActionInfoOutline, 64, 64)
	now, _ := held()

	if was != 1 || now != 1 {
		t.Errorf("the cache holds %d entries after one mask and %d after asking twice", was, now)
	}
}

func TestBothSidesArePartOfTheKey(t *testing.T) {
	empty()

	mask(icons.ActionInfoOutline, 64, 64)
	mask(icons.ActionInfoOutline, 32, 64)

	if got, _ := held(); got != 2 {
		t.Errorf("two shapes of the same icon made %d entries", got)
	}
}

func TestTheCacheStaysUnderItsBudget(t *testing.T) {
	empty()

	for _, side := range []int{1000, 1010, 1020, 1030, 1040, 1050} {
		mask(icons.ActionInfoOutline, side, side)

		if _, bytes := held(); bytes > iconBudget {
			t.Fatalf("holding %d bytes, over the %d budget", bytes, iconBudget)
		}
	}
}

func TestChangingSizeDropsTheOldMasks(t *testing.T) {
	empty()

	set := []Icon{
		icons.ActionInfoOutline,
		icons.ActionSettings,
		icons.AlertWarning,
		icons.ContentAdd,
	}
	for _, i := range set {
		mask(i, 1000, 1000)
	}

	before, _ := held()
	if before != len(set) {
		t.Fatalf("the cache holds %d of %d masks", before, len(set))
	}

	mask(icons.ActionInfoOutline, 1100, 1100)

	entries, bytes := held()
	if bytes > iconBudget {
		t.Errorf("holding %d bytes, over the %d budget", bytes, iconBudget)
	}
	if entries >= before {
		t.Errorf("the cache holds %d entries, want fewer than the %d at the old size",
			entries, before)
	}

	iconMu.Lock()
	defer iconMu.Unlock()
	for at := range iconCache {
		if at.w == 1000 {
			t.Errorf("a mask at the old size is still held: %v", at)
		}
	}
}

func TestTheMaskBeingAskedForSurvivesEviction(t *testing.T) {
	empty()

	for _, side := range []int{900, 950, 1000, 1050} {
		m := mask(icons.ActionInfoOutline, side, side)

		iconMu.Lock()
		kept, ok := iconCache[iconKey{icon: string(icons.ActionInfoOutline), w: side, h: side}]
		iconMu.Unlock()

		if !ok || kept != m {
			t.Fatalf("at %d the mask just made is not the one held", side)
		}
	}
}

func TestTheByteCountMatchesWhatIsHeld(t *testing.T) {
	empty()

	for _, side := range []int{48, 64, 900, 64, 1000} {
		mask(icons.ActionSettings, side, side)
	}

	iconMu.Lock()
	defer iconMu.Unlock()

	var want int
	for _, m := range iconCache {
		want += len(m.Pix)
	}
	if iconBytes != want {
		t.Errorf("the count says %d bytes, the cache holds %d", iconBytes, want)
	}
}

func TestDrawingThroughAnEvictionStillPaints(t *testing.T) {
	empty()
	palette := theme.Default()

	for _, i := range []Icon{icons.ActionInfoOutline, icons.ActionSettings, icons.AlertWarning} {
		mask(i, 1100, 1100)
	}

	img := NewImage(96, 96, palette.Background)
	DrawIcon(img, icons.ActionInfoOutline, Rect{W: 96, H: 96}, palette.Text, palette.Background)

	var lit int
	for y := range 96 {
		for x := range 96 {
			if img.At(x, y) != palette.Background {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("nothing was drawn")
	}

	if _, bytes := held(); bytes > iconBudget {
		t.Errorf("holding %d bytes, over the %d budget", bytes, iconBudget)
	}
}

func TestAMaskLargerThanTheBudgetIsStillDrawn(t *testing.T) {
	empty()
	palette := theme.Default()

	const side = 2200

	img := NewImage(160, 160, palette.Background)
	DrawIconFilling(img, icons.ActionInfoOutline,
		Rect{X: -side/2 + 80, Y: -side/2 + 80, W: side, H: side},
		palette.Text, palette.Background)

	for y := range 160 {
		for x := range 160 {
			if img.At(x, y) != palette.Background {
				return
			}
		}
	}
	t.Error("nothing was drawn")
}
