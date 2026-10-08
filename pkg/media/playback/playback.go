package playback

import (
	"context"
	"io"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/surface"
)

type Branded interface {
	Logo() string
}

type Seeker interface {
	Seek(to time.Duration)
}

type Mark struct {
	From, To time.Duration
	RGB      uint32
}

type Marked interface {
	Marks() []Mark
}

type Target struct {
	Width, Height int
	Tallest       int
	Fastest       int
}

type Waiter interface {
	Waiting(w bool)
}

type Pictures interface {
	Pictured() bool
	Pictures(ctx context.Context) (webm io.ReadCloser, from time.Duration, err error)
}

type Picture struct {
	H264          bool
	Width, Height int
	Next          func(context.Context) (data []byte, at time.Duration, err error)
	Session       uint32
	Sealed        func(context.Context) (data []byte, at time.Duration, crypt *surface.Crypt, err error)
}

type Live interface {
	Live(ctx context.Context) (Picture, error)
}

type Output interface {
	Play(src Source)
	Stop(src Source)
	Handoff(src Source)

	Pause(src Source)
	Resume(src Source)

	Position(src Source) (at time.Duration, paused bool)

	Behind(src Source) time.Duration

	Changed(src Source)

	Attend(s *Session)
	Failed(err error)
}

type Session struct {
	Label    string
	Logo     string
	Pictured bool
}

type Source interface {
	Read(ctx context.Context, pcm []int16) (int, error)

	Start() time.Duration

	Showing() Showing
	Label() string

	Play()
	Pause()
	Stop()
	Next()
	Previous()
}

type Showing struct {
	Title  string
	Artist string
	Album  string

	Art string

	Mark string

	Length time.Duration

	LiveWithin time.Duration

	HasNext, HasPrevious bool

	Upcoming []Upcoming
}

type Upcoming struct {
	Title  string
	Artist string
	Length time.Duration
	Art    string

	Play func()
}
