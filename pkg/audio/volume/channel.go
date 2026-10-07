// Package volume binds controls to a particular audio stream.
package volume

// Channel is a stream's level and its change notifications.
type Channel interface {
	Level() int
	Set(int)
	Watch(func(int)) func()
}

// Service controls independently named audio streams.
type Service[S comparable] interface {
	Level(S) int
	Set(S, int)
	Watch(S, func(int)) func()
}

type channel[S comparable] struct {
	service Service[S]
	stream  S
}

// For binds a volume service to one stream; hardware scaling stays in the service.
func For[S comparable](service Service[S], stream S) Channel {
	return channel[S]{service: service, stream: stream}
}

func (v channel[S]) Level() int                     { return v.service.Level(v.stream) }
func (v channel[S]) Set(level int)                  { v.service.Set(v.stream, level) }
func (v channel[S]) Watch(changed func(int)) func() { return v.service.Watch(v.stream, changed) }

// Stepped is hardware whose saved volume is a discrete step rather than a percentage.
type Stepped interface {
	Step() int
	Set(int)
	WatchVolume(func(int)) func()
}

type stepped struct {
	service Stepped
	count   int
}

// Steps presents a stepped hardware volume as percentages, rounding to the nearest step.
func Steps(service Stepped, count int) Channel {
	if count <= 0 {
		panic("volume: steps must be positive")
	}
	return stepped{service: service, count: count}
}
func (v stepped) Level() int    { return v.service.Step() * 100 / v.count }
func (v stepped) Set(level int) { v.service.Set((min(max(level, 0), 100)*v.count + 50) / 100) }
func (v stepped) Watch(changed func(int)) func() {
	return v.service.WatchVolume(func(step int) { changed(step * 100 / v.count) })
}
