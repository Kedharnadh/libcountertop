package schema

import (
	"github.com/ygelfand/libcountertop/pkg/say"
)

// Stream is a kind of sound, each with its own level.
type Stream string

const (
	StreamMain     Stream = "main"
	StreamMedia    Stream = "media"
	StreamAlerts   Stream = "alerts"
	StreamVoice    Stream = "voice"
	StreamFeedback Stream = "feedback"
)

// Label is how the setting is shown.
func (s Stream) Label() string {
	switch s {
	case StreamMain:
		return "Main"
	case StreamMedia:
		return say.T("stream.media")
	case StreamAlerts:
		return say.T("stream.alerts")
	case StreamVoice:
		return say.T("stream.voice")
	case StreamFeedback:
		return say.T("stream.feedback")
	}
	return string(s)
}
