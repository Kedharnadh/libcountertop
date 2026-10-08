package mark

import (
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

const square = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
	<path d="M6 6h12v12H6z"/>
</svg>`

const dot = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
	<circle cx="12" cy="12" r="6"/>
</svg>`

const curves = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
	<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z"/>
</svg>`

func drawn(t *testing.T, icon ui.Icon, size int) float64 {
	t.Helper()

	palette := theme.Default()
	img := ui.NewImage(size, size, palette.Background)
	ui.DrawIcon(img, icon, ui.Rect{W: size, H: size}, palette.Text, palette.Background)

	var on int
	for y := range size {
		for x := range size {
			if img.At(x, y) != palette.Background {
				on++
			}
		}
	}
	return float64(on) / float64(size*size)
}

func TestAFilledPathDraws(t *testing.T) {
	icon, err := Convert([]byte(square))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}
	if len(icon) == 0 {
		t.Fatal("the icon is empty")
	}

	if got := drawn(t, icon, 48); got < 0.2 || got > 0.3 {
		t.Errorf("the square covers %.2f of the picture, want about a quarter", got)
	}
}

func TestACircleDraws(t *testing.T) {
	icon, err := Convert([]byte(dot))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}

	if got := drawn(t, icon, 48); got < 0.14 || got > 0.24 {
		t.Errorf("the circle covers %.2f of the picture, want about a fifth", got)
	}
}

func TestCurvesDraw(t *testing.T) {
	icon, err := Convert([]byte(curves))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}
	if got := drawn(t, icon, 48); got < 0.5 {
		t.Errorf("the disc covers %.2f of the picture, want most of it", got)
	}
}

func TestATallBoxIsCentredNotStretched(t *testing.T) {
	wide := `<svg viewBox="0 0 24 12"><path d="M0 0h24v12H0z"/></svg>`

	icon, err := Convert([]byte(wide))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}

	if got := drawn(t, icon, 48); got < 0.4 || got > 0.6 {
		t.Errorf("the band covers %.2f of the picture, want about half", got)
	}
}

func TestAnSVGWithNoPathsIsRejected(t *testing.T) {
	_, err := Convert([]byte(`<svg viewBox="0 0 24 24"></svg>`))
	if err == nil {
		t.Fatal("an svg with nothing in it converted")
	}
	if !strings.Contains(err.Error(), "filled paths") {
		t.Errorf("the error is %q, which does not say what was wrong", err)
	}
}

func TestRubbishIsRejected(t *testing.T) {
	if _, err := Convert([]byte("not an svg at all")); err == nil {
		t.Error("something that is not an svg converted")
	}
}

func TestABadViewBoxIsRejected(t *testing.T) {
	_, err := Convert([]byte(`<svg viewBox="0 0 nonsense 24"><path d="M0 0h1v1H0z"/></svg>`))
	if err == nil {
		t.Error("a viewBox that is not numbers converted")
	}
}

func TestNoViewBoxIsTakenAsTwentyFour(t *testing.T) {
	icon, err := Convert([]byte(`<svg><path d="M6 6h12v12H6z"/></svg>`))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}
	if got := drawn(t, icon, 48); got < 0.2 || got > 0.3 {
		t.Errorf("the square covers %.2f of the picture, want about a quarter", got)
	}
}

func TestShapedKeepsTheBoxItWasDrawnIn(t *testing.T) {
	wide := `<svg viewBox="0 0 24 12"><path d="M0 0h24v12H0z"/></svg>`

	icon, err := Shaped([]byte(wide))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}

	if got := drawn(t, icon, 48); got < 0.9 {
		t.Errorf("the band covers %.2f of the picture, want all of it", got)
	}
}

func TestShapedAndConvertAgreeOnASquare(t *testing.T) {
	square := `<svg viewBox="0 0 24 24"><path d="M4 4h16v16H4z"/></svg>`

	plain, err := Convert([]byte(square))
	if err != nil {
		t.Fatalf("converting: %v", err)
	}
	shaped, err := Shaped([]byte(square))
	if err != nil {
		t.Fatalf("converting shaped: %v", err)
	}

	if string(plain) != string(shaped) {
		t.Error("a square drawing converted differently with and without its own box")
	}
}
