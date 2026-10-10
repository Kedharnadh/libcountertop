package call

import (
	"context"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/media/callstatus"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Size = callstatus.Size

type Settings interface {
	Read() schema.Call
	Update(func(*schema.Call)) error
}

type Peers interface {
	Self() peer.Peer
	Find(string) (peer.Peer, bool)
	Touch(string)
}

type Privacy interface {
	MicMuted() bool
	CameraCovered() bool
	Watch(func()) func()
}

type Media interface{ Pause() }

type Messages interface {
	Info(title, body string, hold time.Duration)
}

type Stage interface {
	Hold() (release func())
	Redraw()
}

type Video interface {
	Start(Feed) (Picture, rtc.Camera)
}

type Feed struct {
	Stream schema.CallStream
	Shown  Shown
	Key    func()
	Sized  func(Size)
}

type Shown struct {
	Remote, Own      Size
	RemoteOn, SelfOn bool
}

type Picture interface {
	rtc.Screen
	Update(Shown)
	Stats() (remote, self LayerStats)
	Close()
}

type LayerStats struct {
	Open   bool
	Width  int
	Height int
	Frames uint64
	Failed error
}

type Sound interface {
	rtc.Audio
	Queued() int
	Claim(string, func(context.Context) error)
	Duck(bool)
	Chime(schema.Stream, float64, ...tone.Note)
}

type Dependencies struct {
	Settings Settings
	Peers    Peers
	Port     int
	Privacy  Privacy
	Media    Media
	Sound    Sound
	Stage    Stage
	Video    Video
	Messages Messages
}
