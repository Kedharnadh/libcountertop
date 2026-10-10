package gpu

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
)

func TestAFullScreenLayerIsPlacedUnturnedInEveryOrientation(t *testing.T) {
	const fw, fh = 600, 1024
	for _, o := range []geometry.Orientation{geometry.Rotate0, geometry.Rotate90, geometry.Rotate180, geometry.Rotate270} {
		w, h := o.Size(fw, fh)
		l := &Layer{W: w, H: h, turn: turnFor(o, w, h)}

		x, y, m, _ := o.Place(fw, fh, geometry.Rect{W: w, H: h}, w, h)
		x, y, m = l.placed(x, y, m)
		if x != 0 || y != 0 || m != (surface.Matrix{DsDx: 1, DsDy: 1}) {
			t.Errorf("%v: placed at (%v,%v) with %+v", o, x, y, m)
		}
		if o != geometry.Rotate0 && (l.turn.W != fw || l.turn.H != fh) {
			t.Errorf("%v: window %dx%d, want the panel's %dx%d", o, l.turn.W, l.turn.H, fw, fh)
		}
	}
}

func TestTheTurnShowsEachLayerPixelWhereTheScreenWouldHave(t *testing.T) {
	const w, h = 1024, 600
	o := geometry.Rotate90
	tu := turnFor(o, w, h)
	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {517, 233}} {
		wx, wy := o.Project(tu.W, tu.H, p[0], p[1])
		cx, cy := float32(wx)+0.5, float32(wy)+0.5
		bx, by := tu.A*cx+tu.B*cy+tu.C, tu.D*cx+tu.E*cy+tu.F
		if bx != float32(p[0])+0.5 || by != float32(p[1])+0.5 {
			t.Errorf("layer pixel %v shows at window %d,%d, which samples %v,%v", p, wx, wy, bx, by)
		}
	}
}
