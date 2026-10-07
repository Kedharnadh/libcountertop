package timezone

import (
	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	tz "github.com/ygelfand/libcountertop/pkg/timezone"
	"log/slog"
)

const FollowHome = "Home Assistant"

type PolicyOptions struct {
	Read         func() schema.Time
	Home, Chosen func(string) error
}

// Policy applies a persisted override ahead of the server's timezone.
type Policy struct {
	options PolicyOptions
	Select  *esphome.Select
}

func NewPolicy(options PolicyOptions) *Policy {
	p := &Policy{options: options}
	p.Select = &esphome.Select{Base: esphome.Base{ObjectID: "time_zone", Name: "Time zone", Icon: "mdi:map-clock", Category: esphome.CategoryConfig}, Options: append([]string{FollowHome}, tz.Names()...)}
	p.Select.OnCommand = func(name string) {
		if name == FollowHome {
			name = ""
		}
		p.SetZone(name)
	}
	return p
}

func Label(chosen string) string {
	if chosen == "" {
		return FollowHome
	}
	return chosen
}

func Spec(c schema.Time) string {
	if c.Chosen != "" {
		if z, ok := tz.Named(c.Chosen); ok {
			return z.Spec
		}
		slog.Warn("the chosen time zone is not one we know", "zone", c.Chosen)
	}
	return c.Home
}

func (p *Policy) Restore() {
	c := p.options.Read()
	p.Select.Set(Label(c.Chosen))
	if err := tz.Use(Spec(c)); err != nil {
		slog.Warn("could not use the saved time zone", "err", err)
	}
}

func (p *Policy) Zone() string { return p.options.Read().Chosen }

func (p *Policy) SetZone(name string) {
	if name != "" {
		if _, ok := tz.Named(name); !ok {
			slog.Warn("no such time zone", "zone", name)
			return
		}
	}
	if err := p.options.Chosen(name); err != nil {
		slog.Error("saving the time zone failed", "err", err)
	}
	p.Restore()
}

func (p *Policy) FromHome(said string) {
	if said == "" {
		return
	}
	if said != p.options.Read().Home {
		if err := p.options.Home(said); err != nil {
			slog.Error("saving the time zone failed", "err", err)
		}
	}
	p.Restore()
}
