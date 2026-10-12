package radar

import (
	"image"
	"testing"
)

func TestOutlinesDecode(t *testing.T) {
	for name, set := range map[string][]Outline{"states": States(), "counties": Counties()} {
		if len(set) == 0 {
			t.Fatalf("%s: no outlines", name)
		}
		points := 0
		for _, o := range set {
			points += len(o.Points)
			if o.South > o.North || o.West > o.East {
				t.Fatalf("%s: inverted bounds %+v", name, o)
			}
		}
		if points < 50000 {
			t.Errorf("%s: %d points, want the 1:5M detail", name, points)
		}
	}
}

func TestMassachusettsIsInABostonView(t *testing.T) {
	v := View{Lat: 42.36, Lon: -71.06, Zoom: 8.8, Size: image.Pt(1280, 800)}
	south, west, north, east := v.Bounds()
	if south >= 42.36 || north <= 42.36 || west >= -71.06 || east <= -71.06 {
		t.Fatalf("bounds %v %v %v %v do not hold the center", south, west, north, east)
	}
	inside := 0
	for _, o := range States() {
		if o.Touches(v) {
			inside++
		}
	}
	if inside == 0 || inside > 20 {
		t.Errorf("%d state rings touch a Boston view, want a handful", inside)
	}
}
