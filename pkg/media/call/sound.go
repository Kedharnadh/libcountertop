package call

import (
	"context"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	ringEvery     = 3 * time.Second
	ringLevel     = 0.5
	ringbackEvery = 4 * time.Second
	ringbackLevel = 0.15
	maxQueued     = rtc.PlayRate / 5
)

type conversation struct{ Sound }

func (c conversation) Play(pcm []int16) {
	if c.Queued() <= maxQueued {
		c.Sound.Play(pcm)
	}
}

var (
	ringTone = []tone.Note{
		{Freq: 1047, Ms: 110},
		{Freq: 1319, Ms: 110},
		{Freq: 1047, Ms: 110},
		{Freq: 1319, Ms: 110},
	}
	ringbackTone = []tone.Note{{Freq: 440, Ms: 400}, {Ms: 200}, {Freq: 440, Ms: 400}}
)

func ring(ctx context.Context, sound Sound) bool {
	sound.Duck(true)
	defer sound.Duck(false)
	over := time.NewTimer(ringFor)
	defer over.Stop()
	repeat := time.NewTicker(ringEvery)
	defer repeat.Stop()
	for ctx.Err() == nil {
		sound.Chime(schema.StreamAlerts, ringLevel, ringTone...)
		select {
		case <-ctx.Done():
			return false
		case <-over.C:
			return true
		case <-repeat.C:
		}
	}
	return false
}

func ringback(ctx context.Context, sound Sound) {
	repeat := time.NewTicker(ringbackEvery)
	defer repeat.Stop()
	for ctx.Err() == nil {
		sound.Chime(schema.StreamVoice, ringbackLevel, ringbackTone...)
		select {
		case <-ctx.Done():
			return
		case <-repeat.C:
		}
	}
}
