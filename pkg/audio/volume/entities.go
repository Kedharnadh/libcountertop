package volume

import (
	"log/slog"
	"slices"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Levels interface {
	Level(schema.Stream) int
	Set(schema.Stream, int)
}
type Entities struct {
	Numbers map[schema.Stream]*esphome.Number
	Duck    *esphome.Number
	streams []schema.Stream
	duck    storage.Section[schema.Media]
}

func NewEntities(levels Levels, streams []schema.Stream, duck storage.Section[schema.Media], deviceID uint32) *Entities {
	e := &Entities{Numbers: map[schema.Stream]*esphome.Number{}, streams: slices.Clone(streams), duck: duck}
	for _, s := range streams {
		n := &esphome.Number{Base: esphome.Base{ObjectID: "volume_" + string(s), Name: s.Label() + " volume", Icon: "mdi:volume-high", Category: esphome.CategoryConfig, DeviceID: deviceID}, Min: 0, Max: 100, Step: 1, Unit: "%", Mode: esphome.NumberSlider}
		n.OnCommand = func(level float32) { levels.Set(s, int(level)) }
		e.Numbers[s] = n
	}
	if duck != nil {
		e.Duck = NewDuck(duck, deviceID)
	}
	return e
}
func (e *Entities) Entities() []esphome.Entity {
	out := make([]esphome.Entity, 0, len(e.streams)+1)
	for _, s := range e.streams {
		out = append(out, e.Numbers[s])
	}
	if e.Duck != nil {
		out = append(out, e.Duck)
	}
	return out
}
func (e *Entities) RestoreDuck() {
	if e.Duck != nil {
		e.Duck.Set(float32(e.duck.Read().DuckDB))
	}
}
func (e *Entities) SetDuckDB(db float64) error {
	err := e.duck.Update(func(c *schema.Media) { c.DuckDB = db })
	e.RestoreDuck()
	return err
}

// NewDuck constructs the ducking control owned by one playback component.
func NewDuck(settings storage.Section[schema.Media], deviceID uint32) *esphome.Number {
	n := &esphome.Number{Base: esphome.Base{ObjectID: "media_duck_level", Name: "Music ducking", Icon: "mdi:volume-medium", Category: esphome.CategoryConfig, DeviceID: deviceID}, Min: ducking.MinimumDB, Max: ducking.MaximumDB, Step: 1, Unit: "dB", Mode: esphome.NumberBox}
	n.OnCommand = func(db float32) {
		err := settings.Update(func(c *schema.Media) { c.DuckDB = float64(db) })
		n.Set(float32(settings.Read().DuckDB))
		if err != nil {
			slog.Error("saving the ducking level failed", "err", err)
		}
	}
	return n
}
