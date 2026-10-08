package geometry

type Orientation int

const (
	Rotate0 Orientation = 0

	Rotate90  Orientation = 90
	Rotate180 Orientation = 180
	Rotate270 Orientation = 270
)

func (o Orientation) String() string {
	switch o {
	case Rotate0:
		return "portrait"
	case Rotate90:
		return "landscape"
	case Rotate180:
		return "portrait inverted"
	case Rotate270:
		return "landscape inverted"
	}
	return "unknown"
}

func (o Orientation) Sideways() bool { return o == Rotate90 || o == Rotate270 }

func (o Orientation) Size(fbW, fbH int) (w, h int) {
	if o.Sideways() {
		return fbH, fbW
	}
	return fbW, fbH
}

func (o Orientation) Project(fbW, fbH, vx, vy int) (x, y int) {
	switch o {
	case Rotate90:
		return vy, fbH - 1 - vx
	case Rotate180:
		return fbW - 1 - vx, fbH - 1 - vy
	case Rotate270:
		return fbW - 1 - vy, vx
	}
	return vx, vy
}

func (o Orientation) Unproject(fbW, fbH, x, y int) (vx, vy int) {
	switch o {
	case Rotate90:
		return fbH - 1 - y, x
	case Rotate180:
		return fbW - 1 - x, fbH - 1 - y
	case Rotate270:
		return y, fbW - 1 - x
	}
	return x, y
}
