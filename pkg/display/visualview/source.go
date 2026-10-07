package visualview

import (
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func From(x visual.Input, s config.Source) visual.Input {
	switch s {
	case config.SourceMic:
		x.Speaker, x.SpeakerLeft, x.SpeakerRight = x.Mic, x.MicLeft, x.MicRight
		x.Replying = false
	case config.SourceSpeaker:
		x.Mic, x.MicLeft, x.MicRight = x.Speaker, x.SpeakerLeft, x.SpeakerRight
		x.Replying = true
	}
	return x
}
