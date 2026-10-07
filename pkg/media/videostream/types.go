package videostream

import (
	"context"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
)

type Frame struct {
	Data  []byte
	At    time.Duration
	Crypt *surface.Crypt
}

type Source interface {
	Next(context.Context) (Frame, error)
}

type Clock func() (at time.Duration, running bool)

type Stream struct {
	Codec         uint32
	Width, Height uint32
	Session       uint32
	Source        Source
	Clock         Clock
	Over          Overlay
	Started       func()
	Waiting       func(bool)
}

type Band struct {
	Pix    []byte
	Stride int
	At     geometry.Rect
}

type Overlay interface {
	Band(o geometry.Orientation, fw, fh int) (Band, bool)
	Changed() <-chan struct{}
}

type Report struct {
	Shown, Dropped int
	Turns          int
	Orientation    geometry.Orientation
	At             geometry.Rect
}
