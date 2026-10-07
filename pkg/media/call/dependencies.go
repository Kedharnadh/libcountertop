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

// Settings reads call preferences and atomically changes them in the saved configuration.
type Settings interface {
	Read() schema.Call
	Update(func(*schema.Call)) error
}

// Peers is the discovery directory used to route calls and keep active peers fresh.
type Peers interface {
	Self() peer.Peer
	Find(string) (peer.Peer, bool)
	Touch(string)
}

// Privacy reports physical microphone and camera controls.
type Privacy interface {
	MicMuted() bool
	CameraCovered() bool
	Watch(func()) func()
}

// Media is the playback that yields to a call.
type Media interface{ Pause() }

// Messages presents an informational notification on the device.
type Messages interface {
	Info(title, body string, hold time.Duration)
}

// Display provides the native video surface and its current geometry.
type Display interface {
	Helper() *surface.Client
	Orientation() geometry.Orientation
	Native() (int, int)
}

// Sound is the hardware audio access used for conversation and call tones.
type Sound interface {
	rtc.Audio
	Queued() int
	Claim(string, func(context.Context) error)
	Sounding()
	Duck(bool)
	Chime(float64, ...tone.Note)
}

// Dependencies binds the call controller to device services. Call behavior lives here;
// only hardware access and application service lifetimes belong to the caller.
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
