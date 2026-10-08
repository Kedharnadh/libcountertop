package live

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
	"github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/hook"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	FPS         = 30
	Keyframe    = 2
	Wait        = 2 * time.Second
	Settle      = 2 * time.Second
	reviveEvery = 30 * time.Second
	noTurn      = -1
)

type Options struct {
	Vendor  Vendor
	Streams Streams
	Mirror  bool

	StillOnKeyframe bool

	Turn    func() int
	Open    func(helper.Config) (session.Transport, error)
	Muted   func() bool
	Restart func() error

	Saved  func() schema.Camera
	Writer func() schema.CameraWriter
	Device uint32
}

type Camera struct {
	StreamsChanged hook.Hook[[]int]

	options Options
	hub     *session.Hub

	built sync.Once
	table *setting.Table[Knobs]

	knobsOnce sync.Once
	knobs     *setting.Controls[Knobs]

	reviving sync.Mutex
	revived  time.Time
}

func New(o Options) *Camera {
	c := &Camera{options: o}
	c.hub = session.New(session.Options{
		Snapshot: c.snapshot,
		Open:     o.Open,
		Muted:    o.Muted,
		Recover:  c.revive,
		Wait:     Wait,
		Settle:   Settle,
	})
	return c
}

func (c *Camera) turn() int {
	if c.options.Turn == nil {
		return noTurn
	}
	return c.options.Turn()
}

func (c *Camera) sizesFor(q int) []session.Size {
	k := c.Saved()
	mw, mh := parseSize(k.MainSize)
	out := []session.Size{{Width: mw, Height: mh}}
	if k.SubOn {
		sw, sh := subFor(k)
		out = append(out, session.Size{Width: sw, Height: sh})
	}
	if q >= 0 && q%2 == 1 {
		for i := range out {
			out[i].Width, out[i].Height = out[i].Height, out[i].Width
		}
	}
	return out
}

func (c *Camera) snapshot() session.Snapshot {
	q := c.turn()
	k := c.Saved()
	sizes := c.sizesFor(q)
	cfg := helper.Config{
		Width:    sizes[0].Width,
		Height:   sizes[0].Height,
		FPS:      FPS,
		Bitrate:  bitrateFor(k, sizes[0].Width, sizes[0].Height),
		Keyframe: k.Keyframe,
		Params:   c.Params(k),
		Turn:     q,
		Mirror:   c.options.Mirror,
	}
	if len(sizes) > 1 {
		cfg.SubWidth = sizes[1].Width
		cfg.SubHeight = sizes[1].Height
		cfg.SubBitrate = bitrateFor(k, sizes[1].Width, sizes[1].Height)
	}
	s := session.Snapshot{Config: cfg, Key: fmt.Sprintf("%d:%s", q, shapeOf(k))}
	if c.options.StillOnKeyframe {
		s.StillExtra = time.Duration(k.Keyframe) * time.Second
	}
	return s
}

func (c *Camera) Sessions() *session.Hub { return c.hub }

func (c *Camera) Sizes() []session.Size { return c.hub.Sizes() }

func (c *Camera) Bitrate(w, h int) int { return bitrateFor(c.Saved(), w, h) }

func (c *Camera) Still(within time.Duration) (session.Picture, error) {
	return c.hub.Still(within)
}

func (c *Camera) revive(cause error) {
	c.reviving.Lock()
	defer c.reviving.Unlock()
	if time.Since(c.revived) < reviveEvery || c.options.Restart == nil {
		return
	}
	c.revived = time.Now()
	slog.Warn("restarting the camera helper", "err", cause)
	if err := c.options.Restart(); err != nil {
		slog.Error("restarting the camera helper failed", "err", err)
	}
}

func (c *Camera) Saved() Knobs {
	k, bad := c.Table().Configured(c.Defaults(), c.options.Saved().Settings)
	for _, err := range bad {
		slog.Warn("a saved camera setting no longer applies", "err", err)
	}
	return k
}

func (c *Camera) Set(name, value string) error {
	if err := c.options.Writer().Set(name, value); err != nil {
		return err
	}
	c.Apply()
	if name == "main_on" || name == "sub_on" {
		c.StreamsChanged.Emit(c.Served())
	}
	return nil
}

func (c *Camera) Reset() error {
	if err := c.options.Writer().Clear(); err != nil {
		return err
	}
	c.Apply()
	c.controls().Publish()
	c.StreamsChanged.Emit(c.Served())
	return nil
}

func (c *Camera) Apply() {
	if err := c.hub.SetParams(c.Params(c.Saved())); err != nil {
		slog.Warn("the camera settings could not be applied", "err", err)
	}
}

func (c *Camera) controls() *setting.Controls[Knobs] {
	c.knobsOnce.Do(func() {
		c.knobs = &setting.Controls[Knobs]{
			Table:  c.Table(),
			Device: c.options.Device,
			Want:   func(setting.Group) bool { return true },
			Read:   c.Saved,
			Save:   func(s Knob, v string) error { return c.Set(s.Name, v) },
		}
	})
	return c.knobs
}

func (c *Camera) Name() string { return "camera" }

func (c *Camera) Entities() []esphome.Entity {
	reset := &esphome.Button{
		Base: esphome.Base{
			ObjectID: "camera_reset",
			Name:     "Reset camera settings",
			Icon:     "mdi:restore",
			Category: esphome.CategoryConfig,
			DeviceID: c.options.Device,
		},
		OnPress: func() {
			if err := c.Reset(); err != nil {
				slog.Error("resetting the camera settings", "err", err)
			}
		},
	}
	return append(c.controls().Entities(), reset)
}

func (c *Camera) Publish() { c.controls().Publish() }
