package vision

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/camera/card"
	"github.com/ygelfand/libcountertop/pkg/camera/session"
)

const (
	Wait       = 5 * time.Second
	Quality    = 85
	cardWidth  = 640
	cardHeight = 360
)

type Options struct {
	DeviceID uint32
	Covered  func() bool
	Picture  func(within time.Duration) (session.Picture, error)
}

type Vision struct {
	entity  *esphome.Camera
	covered func() bool
	picture func(within time.Duration) (session.Picture, error)

	mu sync.Mutex
}

func New(o Options) *Vision {
	v := &Vision{covered: o.Covered, picture: o.Picture}
	v.entity = &esphome.Camera{
		Base: esphome.Base{
			ObjectID: "camera",
			DeviceID: o.DeviceID,
			Name:     "Camera",
			Icon:     "mdi:cctv",
		},
		Image: v.Still,
	}
	return v
}

func (v *Vision) Name() string { return "vision" }

func (v *Vision) Entities() []esphome.Entity { return []esphome.Entity{v.entity} }

func (v *Vision) Still() ([]byte, error) {
	if v.covered() {
		return card.JPEG(cardWidth, cardHeight, session.Shuttered, Quality)
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	pic, err := v.picture(Wait)
	switch {
	case errors.Is(err, session.ErrMuted):
		return card.JPEG(cardWidth, cardHeight, session.Muted, Quality)
	case errors.Is(err, session.ErrShutter):
		return card.JPEG(cardWidth, cardHeight, session.Shuttered, Quality)
	case err != nil:
		return nil, err
	}
	return encode(pic)
}

func encode(pic session.Picture) ([]byte, error) {
	w, h := pic.Width, pic.Height
	if w <= 0 || h <= 0 || len(pic.RGBA) < w*h*4 {
		return nil, errors.New("vision: a short picture")
	}
	img := &image.RGBA{Pix: pic.RGBA, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: Quality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
