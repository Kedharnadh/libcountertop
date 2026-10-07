// Package activity is what a turn leaves behind in Home Assistant's logbook.
//
// The phases are already there: Home Assistant's own assist_satellite entity moves through idle,
// listening, processing and responding, and those state changes show in the feed. What it cannot
// show is content or which assistant answered, because it does not know either. These do.
package activity

import (
	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/libcountertop/pkg/runtime/component"
)

// Log is the three sensors. It has no hardware and no loop: something else has a turn, and tells it.
type Log struct {
	opts  Options
	word  *esphome.TextSensor
	heard *esphome.TextSensor
	reply *esphome.TextSensor
}

// Options connects the recorder to the product event bus.
type Options struct {
	Emit func(map[string]string)
}

// New creates the activity sensors and turn recorder.
func New(opts Options) *Log {
	return &Log{opts: opts,
		word:  &esphome.TextSensor{Base: esphome.Base{ObjectID: "last_wake_word", Name: "Last wake word", Icon: "mdi:account-voice"}},
		heard: &esphome.TextSensor{Base: esphome.Base{ObjectID: "last_heard", Name: "Last heard", Icon: "mdi:ear-hearing"}},
		reply: &esphome.TextSensor{Base: esphome.Base{ObjectID: "last_reply", Name: "Last reply", Icon: "mdi:message-text"}},
	}
}

func (l *Log) Name() string { return "activity" }

func (l *Log) Entities() []esphome.Entity {
	return []esphome.Entity{l.word, l.heard, l.reply}
}

// Woke records which wake word started a turn, so the transcript that follows can be attributed.
func (l *Log) Woke(phrase string) {
	if phrase == "" {
		return
	}
	l.word.Set(component.Fit(phrase))
}

func (l *Log) Heard(text string) {
	if text == "" {
		return
	}
	l.heard.Set(component.Fit(text))
}

func (l *Log) Replied(text string) {
	if text == "" {
		return
	}
	l.reply.Set(component.Fit(text))
}
