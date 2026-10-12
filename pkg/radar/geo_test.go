package radar

import (
	"image"
	"math"
	"testing"
)

func TestWorldPixel(t *testing.T) {
	x, y := worldPixel(0, 0, 0)
	if x != 128 || math.Abs(y-128) > 1e-9 {
		t.Errorf("worldPixel(0,0,0) = %v,%v, want 128,128", x, y)
	}
	x, _ = worldPixel(0, 180, 1)
	if x != 512 {
		t.Errorf("worldPixel(0,180,1) x = %v, want 512", x)
	}
}

func TestZoomForCoversTheRange(t *testing.T) {
	lat, width := 41.3, 1280
	z := zoomFor(lat, 200, width)
	perPixel := equatorMeter * math.Cos(lat*math.Pi/180) / math.Exp2(z)
	if miles := perPixel * float64(width) / metersPerMi; math.Abs(miles-200) > 0.01 {
		t.Errorf("zoom %v spans %v mi, want 200", z, miles)
	}
}

func TestLower48(t *testing.T) {
	for _, c := range []struct {
		lat, lon float64
		want     bool
	}{
		{41.3, -73.06, true},
		{39.1, -94.58, true},
		{61.2, -149.9, false},
		{51.5, -0.12, false},
	} {
		if got := lower48(c.lat, c.lon); got != c.want {
			t.Errorf("lower48(%v,%v) = %v, want %v", c.lat, c.lon, got, c.want)
		}
	}
}

func TestViewPixelCentersTheView(t *testing.T) {
	v := View{Lat: 41.3, Lon: -73.06, Zoom: 8.9, Size: image.Pt(1280, 800)}
	if got := v.Pixel(41.3, -73.06); got != image.Pt(640, 400) {
		t.Errorf("center pixel = %v, want (640,400)", got)
	}
	if got := v.Pixel(41.3, -72.9); got.X <= 640 {
		t.Errorf("a point east of center = %v, want right of 640", got)
	}
}

func TestViewLatLonInvertsPixel(t *testing.T) {
	v := View{Lat: 41.3, Lon: -73.06, Zoom: 8.9, Size: image.Pt(1280, 800)}
	for _, p := range [][2]float64{{41.3, -73.06}, {42.1, -71.5}, {40.2, -75.0}} {
		px := v.Pixel(p[0], p[1])
		lat, lon := v.LatLon(float64(px.X), float64(px.Y))
		if math.Abs(lat-p[0]) > 0.01 || math.Abs(lon-p[1]) > 0.01 {
			t.Errorf("LatLon(Pixel(%v)) = %v,%v", p, lat, lon)
		}
	}
}
