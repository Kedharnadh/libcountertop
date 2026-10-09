package schema

import (
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Stream string

const (
	StreamMain     Stream = "main"
	StreamMedia    Stream = "media"
	StreamAlerts   Stream = "alerts"
	StreamVoice    Stream = "voice"
	StreamFeedback Stream = "feedback"
)

func (s Stream) Label() string {
	switch s {
	case StreamMain:
		return say.T("stream.main")
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
