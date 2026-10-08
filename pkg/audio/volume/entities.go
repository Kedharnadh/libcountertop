package volume

import (
	"slices"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Levels interface {
	Level(schema.Stream) int
	Set(schema.Stream, int)
}
type Entities struct {
	Numbers map[schema.Stream]*esphome.Number
	streams []schema.Stream
}

func NewEntities(
	levels Levels,
	streams []schema.Stream,
	deviceID uint32,
) *Entities {
	e := &Entities{
		Numbers: map[schema.Stream]*esphome.Number{},
		streams: slices.Clone(streams),
	}
	for _, s := range streams {
		n := &esphome.Number{
			Base: esphome.Base{
				ObjectID: "volume_" + string(s),
				Name:     s.Label() + " volume",
				Icon:     "mdi:volume-high",
				Category: esphome.CategoryConfig,
				DeviceID: deviceID,
			},
			Min:  0,
			Max:  100,
			Step: 1,
			Unit: "%",
			Mode: esphome.NumberSlider,
		}
		n.OnCommand = func(level float32) { levels.Set(s, int(level)) }
		e.Numbers[s] = n
	}
	return e
}
func (e *Entities) Entities() []esphome.Entity {
	out := make([]esphome.Entity, 0, len(e.streams))
	for _, s := range e.streams {
		out = append(out, e.Numbers[s])
	}
	return out
}
