package turn

import (
	"context"
	"time"

	"github.com/ygelfand/libcountertop/pkg/assistant/activity"
	"github.com/ygelfand/libcountertop/pkg/inference/wake"
)

// Microphone supplies processed 16 kHz mono audio and its recent history.
type Microphone interface {
	Listen(string) (<-chan []int16, func())
	Recent(time.Duration) []int16
	Gain() float64
	Clipped() uint64
}

type Speaker interface {
	VoiceSink
	Splices() uint64
	Underruns() uint64
	Clipped() uint64
}

type Claim interface {
	Done() <-chan struct{}
	Err() error
	Stopped() bool
	Quiet() time.Time
}

type Sound interface {
	Claim(string, func(context.Context, Speaker) error) Claim
	Busy() bool
	Silence()
}

type Recorder interface {
	Opens(string, int)
	Frame([]byte)
	Closes()
	Seconds(string) float64
}

type Words struct {
	MaxListen, MaxThink, FollowUp, Buffer, ChimeLength func(int) time.Duration
	Delivery                                           func(int) string
	Tones                                              func(int) bool
	Chime                                              func(int)
}

// Options binds the conversation to audio hardware, saved settings, and presentation.
type Options struct {
	Source                                    Microphone
	Speaker                                   Speaker
	Sound                                     Sound
	Player                                    interface{ Sounding(bool) }
	Log                                       *activity.Log
	Recorder                                  Recorder
	Words                                     Words
	Word                                      func(int) (string, bool)
	Models                                    func() []wake.Model
	Muted                                     func() (bool, error)
	Adapting, Duck                            func(bool)
	HardwareTail                              time.Duration
	Shown                                     func(Showing)
	Begin, End, Cancelled, Failure, Elsewhere func()
	Effect                                    func(int, Phase)
}

func dropped(source Microphone) uint64 {
	if s, ok := source.(interface{ Dropped() uint64 }); ok {
		return s.Dropped()
	}
	return 0
}
