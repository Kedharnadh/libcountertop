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

func every(s tone.Sound, least time.Duration) time.Duration {
	return max(least, s.Length()+time.Second)
}

func ring(ctx context.Context, sound Sound, snd tone.Sound) bool {
	sound.Duck(true)
	defer sound.Duck(false)
	over := time.NewTimer(ringFor)
	defer over.Stop()
	repeat := time.NewTicker(every(snd, ringEvery))
	defer repeat.Stop()
	for ctx.Err() == nil {
		sound.Tone(schema.StreamAlerts, ringLevel, snd)
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

func ringback(ctx context.Context, sound Sound, snd tone.Sound) {
	repeat := time.NewTicker(every(snd, ringbackEvery))
	defer repeat.Stop()
	for ctx.Err() == nil {
		sound.Tone(schema.StreamVoice, ringbackLevel, snd)
		select {
		case <-ctx.Done():
			return
		case <-repeat.C:
		}
	}
}
