package volume

import (
	"slices"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Options struct {
	Mixer     MixerOptions
	Main      Channel
	MainSteps int
}

type Volume struct {
	*Mixer

	main      Channel
	mainSteps int
	mainNum   *Entities
}

func NewVolume(o Options) *Volume {
	if o.Main != nil {
		o.Mixer.Streams = slices.DeleteFunc(
			slices.Clone(o.Mixer.Streams),
			func(s schema.Stream) bool { return s == schema.StreamMain },
		)
	}
	v := &Volume{Mixer: NewMixer(o.Mixer), main: o.Main, mainSteps: o.MainSteps}
	if v.main == nil {
		return v
	}
	v.mainNum = NewEntities(v, []schema.Stream{schema.StreamMain}, o.Mixer.Device)
	v.main.Watch(func(level int) {
		v.mainNum.Numbers[schema.StreamMain].Set(float32(level))
		v.Changed.Emit(Change{Stream: schema.StreamMain, Level: level})
	})
	return v
}

func (v *Volume) Entities() []esphome.Entity {
	if v.main == nil {
		return v.Mixer.Entities()
	}
	return append(v.mainNum.Entities(), v.Mixer.Entities()...)
}

func (v *Volume) Restore(level func(schema.Stream) int) {
	v.Mixer.Restore(level)
	if v.main != nil {
		v.mainNum.Numbers[schema.StreamMain].Set(float32(v.main.Level()))
	}
}

func (v *Volume) Level(s schema.Stream) int {
	if s == schema.StreamMain && v.main != nil {
		return v.main.Level()
	}
	return v.Mixer.Level(s)
}

func (v *Volume) Set(s schema.Stream, level int) {
	if s == schema.StreamMain && v.main != nil {
		v.main.Set(Clamp(level))
		return
	}
	v.Mixer.Set(s, level)
}

func (v *Volume) Adjust(s schema.Stream, steps int) {
	size := Step
	if s == schema.StreamMain && v.main != nil && v.mainSteps > 0 {
		size = (100 + v.mainSteps - 1) / v.mainSteps
	}
	v.Set(s, v.Level(s)+steps*size)
}

func (v *Volume) Watch(s schema.Stream, changed func(int)) func() {
	if s == schema.StreamMain && v.main != nil {
		return v.main.Watch(changed)
	}
	return v.Mixer.Watch(s, changed)
}

func (v *Volume) Number(s schema.Stream) *esphome.Number {
	if s == schema.StreamMain && v.main != nil {
		return v.mainNum.Numbers[s]
	}
	return v.Mixer.Number(s)
}

func (v *Volume) Streams() []schema.Stream {
	if v.main == nil {
		return v.Mixer.Streams()
	}
	return append([]schema.Stream{schema.StreamMain}, v.Mixer.Streams()...)
}
