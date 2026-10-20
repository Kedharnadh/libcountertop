package call

import (
	"context"
	"net/http"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/media/callstatus"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
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
	Tone(schema.Stream, float64, tone.Sound)
}

type Tones interface {
	Sound(c, fallback schema.Chime) tone.Sound
	Choices() []setting.Option
}

type Dependencies struct {
	Settings Settings
	Peers    Peers
	Port     int
	Secure   int
	Client   *http.Client
	Privacy  Privacy
	Media    Media
	Sound    Sound
	Tones    Tones
	Stage    Stage
	Video    Video
	Messages Messages
}
