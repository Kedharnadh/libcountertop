package radar

import (
	"image"
	"math"
)

const (
	tileSize     = 256
	metersPerMi  = 1609.344
	equatorMeter = 156543.03392804097
)

func worldPixel(lat, lon, zoom float64) (x, y float64) {
	scale := tileSize * math.Exp2(zoom)
	lat = max(-85.05112878, min(85.05112878, lat))
	s := math.Sin(lat * math.Pi / 180)
	x = (lon + 180) / 360 * scale
	y = (0.5 - math.Log((1+s)/(1-s))/(4*math.Pi)) * scale
	return x, y
}

func zoomFor(lat float64, miles, width int) float64 {
	perPixel := float64(miles) * metersPerMi / float64(max(width, 1))
	return math.Log2(equatorMeter * math.Cos(lat*math.Pi/180) / perPixel)
}

func lower48(lat, lon float64) bool {
	return lat >= 24 && lat <= 50 && lon >= -125.5 && lon <= -66
}

type View struct {
	Lat, Lon float64
	Zoom     float64
	Size     image.Point
}

func (v View) Bounds() (south, west, north, east float64) {
	cx, cy := worldPixel(v.Lat, v.Lon, v.Zoom)
	world := tileSize * math.Exp2(v.Zoom)
	hw, hh := float64(v.Size.X)/2, float64(v.Size.Y)/2
	latAt := func(y float64) float64 {
		return math.Atan(math.Sinh(math.Pi*(1-2*y/world))) * 180 / math.Pi
	}
	west = (cx-hw)/world*360 - 180
	east = (cx+hw)/world*360 - 180
	return latAt(cy + hh), west, latAt(cy - hh), east
}

func (v View) LatLon(x, y float64) (lat, lon float64) {
	cx, cy := worldPixel(v.Lat, v.Lon, v.Zoom)
	world := tileSize * math.Exp2(v.Zoom)
	wx := cx + x - float64(v.Size.X)/2
	wy := cy + y - float64(v.Size.Y)/2
	lon = math.Mod(wx/world*360+360, 360) - 180
	lat = math.Atan(math.Sinh(math.Pi*(1-2*wy/world))) * 180 / math.Pi
	return lat, lon
}

func (v View) Pixel(lat, lon float64) image.Point {
	cx, cy := worldPixel(v.Lat, v.Lon, v.Zoom)
	x, y := worldPixel(lat, lon, v.Zoom)
	world := tileSize * math.Exp2(v.Zoom)
	dx := x - cx
	if dx > world/2 {
		dx -= world
	} else if dx < -world/2 {
		dx += world
	}
	return image.Pt(
		int(math.Round(float64(v.Size.X)/2+dx)),
		int(math.Round(float64(v.Size.Y)/2+y-cy)),
	)
}
