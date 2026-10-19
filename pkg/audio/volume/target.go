package volume

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const Recent = 2 * time.Second

type Claimer interface{ Shows(schema.Stream) bool }

type Aim struct {
	Mode   schema.VolumeButtons
	Picked func() (schema.Stream, bool)
	Screen func() any
	Heard  func(schema.Stream) bool
}

var (
	claims = []schema.Stream{schema.StreamMedia, schema.StreamVoice}
	heard  = []schema.Stream{schema.StreamAlerts, schema.StreamVoice, schema.StreamMedia}
)

func Target(a Aim) schema.Stream {
	if a.Picked != nil {
		if s, ok := a.Picked(); ok {
			return s
		}
	}
	if a.Mode == schema.ButtonsStatic {
		return schema.StreamMain
	}
	if a.Screen != nil {
		if c, ok := a.Screen().(Claimer); ok {
			for _, s := range claims {
				if c.Shows(s) {
					return s
				}
			}
		}
	}
	if a.Heard != nil {
		for _, s := range heard {
			if a.Heard(s) {
				return s
			}
		}
	}
	return schema.StreamMain
}
