package panel

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/geometry"
)

const (
	fbW = 1200
	fbH = 1920
)

func TestSize(t *testing.T) {
	tests := []struct {
		name         string
		rot          geometry.Orientation
		wantW, wantH int
	}{
		{"portrait is the panel itself", geometry.Rotate0, fbW, fbH},
		{"landscape swaps them", geometry.Rotate90, fbH, fbW},
		{"upside down does not", geometry.Rotate180, fbW, fbH},
		{"landscape inverted swaps them", geometry.Rotate270, fbH, fbW},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := tt.rot.Size(fbW, fbH)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("Size() = %dx%d, want %dx%d", w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestProjectStaysInTheFramebuffer(t *testing.T) {
	for _, rot := range []geometry.Orientation{geometry.Rotate0, geometry.Rotate90, geometry.Rotate180, geometry.Rotate270} {
		w, h := rot.Size(fbW, fbH)

		for _, v := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}} {
			x, y := rot.Project(fbW, fbH, v[0], v[1])
			if x < 0 || x >= fbW || y < 0 || y >= fbH {
				t.Errorf(
					"%v: Project(%d, %d) = %d,%d, outside %dx%d",
					rot,
					v[0],
					v[1],
					x,
					y,
					fbW,
					fbH,
				)
			}
		}
	}
}

func TestUnprojectIsTheInverseOfProject(t *testing.T) {
	for _, rot := range []geometry.Orientation{geometry.Rotate0, geometry.Rotate90, geometry.Rotate180, geometry.Rotate270} {
		w, h := rot.Size(fbW, fbH)

		for _, v := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}, {w / 2, h / 2}, {7, 13}} {
			x, y := rot.Project(fbW, fbH, v[0], v[1])
			gx, gy := rot.Unproject(fbW, fbH, x, y)

			if gx != v[0] || gy != v[1] {
				t.Errorf("%v: %d,%d projected to %d,%d and came back %d,%d",
					rot, v[0], v[1], x, y, gx, gy)
			}
		}
	}
}

func TestProjectIsOneToOne(t *testing.T) {
	for _, rot := range []geometry.Orientation{geometry.Rotate0, geometry.Rotate90, geometry.Rotate180, geometry.Rotate270} {
		w, h := rot.Size(fbW, fbH)
		seen := map[[2]int]bool{}

		for vx := 0; vx < w; vx += 7 {
			for vy := 0; vy < h; vy += 11 {
				at := [2]int{}
				at[0], at[1] = rot.Project(fbW, fbH, vx, vy)

				if seen[at] {
					t.Fatalf("%v: two viewed pixels land on %v", rot, at)
				}
				seen[at] = true
			}
		}
	}
}

func TestRotate0IsIdentity(t *testing.T) {
	for _, v := range [][2]int{{0, 0}, {5, 9}, {fbW - 1, fbH - 1}} {
		x, y := geometry.Rotate0.Project(fbW, fbH, v[0], v[1])
		if x != v[0] || y != v[1] {
			t.Errorf("geometry.Rotate0 moved %d,%d to %d,%d", v[0], v[1], x, y)
		}
	}
}

func TestOrientationNames(t *testing.T) {
	for _, tt := range []struct {
		rot  geometry.Orientation
		want string
	}{
		{geometry.Rotate0, "portrait"},
		{geometry.Rotate90, "landscape"},
		{geometry.Rotate180, "portrait inverted"},
		{geometry.Rotate270, "landscape inverted"},
	} {
		if got := tt.rot.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", int(tt.rot), got, tt.want)
		}
	}
}

func TestDriverHoldsOrientationWithNoPanel(t *testing.T) {
	d := NewDriver(
		func() Config { return Config{Framebuffer: "/dev/null", Mounted: geometry.Rotate90} },
	)

	if got := d.Orientation(); got != geometry.Rotate90 {
		t.Errorf("a new driver is at %v, want %v", got, geometry.Rotate90)
	}

	d.SetOrientation(geometry.Rotate180)
	if got := d.Orientation(); got != geometry.Rotate180 {
		t.Errorf("after turning, %v, want %v", got, geometry.Rotate180)
	}
}

func TestSettingTheSameOrientationDoesNothing(t *testing.T) {
	d := NewDriver(nullConfig)
	d.SetOrientation(geometry.Rotate180)

	select {
	case <-d.changed:
	default:
	}

	d.SetOrientation(geometry.Rotate180)

	select {
	case <-d.changed:
		t.Error("setting the same orientation asked for a redraw")
	default:
	}
}

func TestTurnResizesTheViewedPanel(t *testing.T) {
	p := &Panel{fbW: fbW, fbH: fbH}

	p.Turn(geometry.Rotate90)
	if p.Width != fbH || p.Height != fbW {
		t.Errorf("landscape is %dx%d, want %dx%d", p.Width, p.Height, fbH, fbW)
	}

	p.Turn(geometry.Rotate0)
	if p.Width != fbW || p.Height != fbH {
		t.Errorf("portrait is %dx%d, want %dx%d", p.Width, p.Height, fbW, fbH)
	}
}
