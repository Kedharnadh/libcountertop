package widgets

import (
	"github.com/ygelfand/libcountertop/pkg/display/message"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func Alerter(m *message.Messages) (show func(tessera.Alert), clear func()) {
	show = func(a tessera.Alert) {
		keys := make([]message.Key, len(a.Buttons))
		for i, b := range a.Buttons {
			keys[len(a.Buttons)-1-i] = message.Key{Text: b.Text, Swatch: b.Color}
		}
		m.Show(message.Message{
			Title:  a.Title,
			Body:   a.Subtitle,
			Tone:   schema.ToneInfo,
			Icon:   a.Icon,
			Swatch: a.Color,
			Keys:   keys,
			Flash:  a.Flash,
			Ended: func(e message.End) {
				if a.Ended != nil {
					a.Ended(alertAction(e, len(keys)))
				}
			},
		}, a.Timeout)
	}
	return show, m.Hide
}

func alertAction(e message.End, keys int) string {
	switch e.Why {
	case message.EndKey:
		if keys-1-e.Key == 0 {
			return tessera.AlertOK
		}
		return tessera.AlertSecond
	case message.EndTimeout:
		return tessera.AlertTimeout
	case message.EndReplaced:
		return tessera.AlertReplaced
	}
	return tessera.AlertRemote
}
