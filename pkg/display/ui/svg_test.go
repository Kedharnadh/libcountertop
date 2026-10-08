package ui

import (
	"image"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

const ring = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <circle cx="12" cy="12" r="9" fill="none" stroke="black" stroke-width="2"/>
</svg>`

const block = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <rect x="4" y="4" width="16" height="16" fill="black"/>
</svg>`

func TestTheTwoFormatsAreToldApartByTheirFirstBytes(t *testing.T) {
	if !IsVG(Icon{0x89, 'I', 'V', 'G', 0x02, 0x0a}) {
		t.Error("an iconvg icon was not recognised as one")
	}
	if IsVG(Icon(ring)) {
		t.Error("an svg icon was taken for iconvg")
	}
	if IsVG(nil) {
		t.Error("nothing was taken for iconvg")
	}
}

func TestAStrokedIconKeepsItsHole(t *testing.T) {
	const side = 48

	m := mask(Icon(ring), side, side)

	middle := m.Pix[side/2*m.Stride+side/2]
	if middle != 0 {
		t.Errorf("the centre of a stroked ring has %d coverage, want none", middle)
	}

	var found bool
	for x := 2; x < 12; x++ {
		if m.Pix[side/2*m.Stride+x] > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("the ring's stroke drew nothing")
	}
}

func TestAFilledIconCoversItsMiddle(t *testing.T) {
	const side = 48

	m := mask(Icon(block), side, side)
	if got := m.Pix[side/2*m.Stride+side/2]; got != 0xff {
		t.Errorf("the centre of a filled square has %d coverage, want full", got)
	}
}

func TestRubbishDrawsNothing(t *testing.T) {
	const side = 16

	m := mask(Icon("<svg>this is not finished"), side, side)
	if m == nil {
		t.Fatal("a broken icon gave no mask at all")
	}
	for i, v := range m.Pix {
		if v != 0 {
			t.Fatalf("byte %d of a broken icon's mask is %d", i, v)
		}
	}
}

func TestEachSizeIsItsOwn(t *testing.T) {
	small := mask(Icon(block), 16, 16)
	large := mask(Icon(block), 64, 64)

	if len(small.Pix) == len(large.Pix) {
		t.Errorf("both sizes came back as %d bytes", len(small.Pix))
	}
	if again := mask(Icon(block), 64, 64); again != large {
		t.Error("the same icon at the same size was rasterized twice")
	}
}

const twoColor = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <rect x="0" y="0" width="12" height="24" fill="#ff0000"/>
  <rect x="12" y="0" width="12" height="24" fill="#0000ff"/>
</svg>`

func TestALogoKeepsItsColors(t *testing.T) {
	const side = 24

	s := NewImage(side, side, theme.Color{})
	DrawColored(s, Icon(twoColor), Rect{W: side, H: side}, theme.Color{})

	left := s.At(side/4, side/2)
	right := s.At(side*3/4, side/2)

	if left.R < 0xc0 || left.G > 0x40 || left.B > 0x40 {
		t.Errorf("the left half came out %+v, want red", left)
	}
	if right.B < 0xc0 || right.R > 0x40 || right.G > 0x40 {
		t.Errorf("the right half came out %+v, want blue", right)
	}
}

func TestTheSameIconAsCoverageLosesThem(t *testing.T) {
	const side = 24

	m := mask(Icon(twoColor), side, side)
	if got := m.Pix[side/2*m.Stride+side/4]; got != 0xff {
		t.Errorf("the left half has %d coverage, want full", got)
	}
	if got := m.Pix[side/2*m.Stride+side*3/4]; got != 0xff {
		t.Errorf("the right half has %d coverage, want full", got)
	}
}

func TestBothCachesAreEvictedTogether(t *testing.T) {
	iconMu.Lock()
	iconCache, colorCache, iconBytes = map[iconKey]*image.Alpha{}, map[iconKey]*image.RGBA{}, 0
	iconMu.Unlock()

	mask(Icon(block), 32, 32)
	inColor(Icon(twoColor), 32, 32)

	iconMu.Lock()
	before := iconBytes
	iconMu.Unlock()
	if before == 0 {
		t.Fatal("nothing was counted")
	}

	iconMu.Lock()
	evict(iconKey{w: 64, h: 64})
	after, alphas, colors := iconBytes, len(iconCache), len(colorCache)
	iconMu.Unlock()

	if alphas != 0 || colors != 0 {
		t.Errorf("%d coverage and %d color entries survived a size change", alphas, colors)
	}
	if after != 0 {
		t.Errorf("%d bytes still counted after both caches were emptied", after)
	}
}
