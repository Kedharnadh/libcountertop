package wake

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"
)

type CacheEntities struct {
	lib    *Library
	inUse  func() []string
	cached *esphome.Sensor
	purge  *esphome.Button
}

func (l *Library) CacheEntities(inUse func() []string) *CacheEntities {
	c := &CacheEntities{lib: l, inUse: inUse}
	c.cached = &esphome.Sensor{
		Base: esphome.Base{
			ObjectID: "cached_data",
			Name:     "Cached data",
			Icon:     "mdi:cached",
			Category: esphome.CategoryDiagnostic,
		},
		Unit:        "kB",
		DeviceClass: "data_size",
		StateClass:  esphome.StateClassMeasurement,
	}
	c.purge = &esphome.Button{
		Base: esphome.Base{
			ObjectID: "purge_cache",
			Name:     "Purge cache",
			Icon:     "mdi:delete-sweep",
			Category: esphome.CategoryDiagnostic,
		},
		OnPress: func() {
			gone, freed := l.Purge(c.inUse())
			slog.Info("cache purged", "models", gone, "bytes", freed)
			c.Measure()
		},
	}
	return c
}

func (c *CacheEntities) Entities() []esphome.Entity { return []esphome.Entity{c.cached, c.purge} }

func (c *CacheEntities) Measure() {
	_, bytes := Cached(c.lib.Dir(), c.inUse())
	c.cached.Set(float32(bytes / 1024))
}
