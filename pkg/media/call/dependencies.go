package call

import (
	"context"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
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

type Display interface {
	Helper() *surface.Client
	Orientation() geometry.Orientation
	Native() (int, int)
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
	Shell    *shell.Shell
	Privacy  Privacy
	Media    Media
	Sound    Sound
	Camera   *Camera
	Display  Display
	Messages Messages
}
