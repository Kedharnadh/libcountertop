package turn

import (
	"context"
	"github.com/ygelfand/libcountertop/pkg/media/pcm"
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

// announce plays what Home Assistant asks for, then reports back. It runs off the connection's
// read loop: fetching and playing must not block it.
func (t *Conversation) announce(a esphome.Announce) {
	slog.Info("announce", "text", a.Text, "start_conversation", a.StartConversation)

	// One claim covers both urls, so silencing an announcement stops the whole thing rather than
	// letting the second one start once the first has been drained.
	t.player.Sounding(true)
	claim := t.sound.Claim("announce", func(ctx context.Context, _ Speaker) error {
		for _, url := range []string{a.PreannounceMediaID, a.MediaID} {
			if url == "" {
				continue
			}
			if err := t.play(ctx, url, nil); err != nil {
				return err
			}
		}
		return nil
	})

	safe.Go("announce", func() {
		<-claim.Done()
		t.player.Sounding(false)

		if err := claim.Err(); err != nil {
			slog.Error("playing the announcement failed", "err", err)
		}
		if err := t.vs.AnnounceFinished(claim.Err() == nil && !claim.Stopped()); err != nil {
			slog.Error("reporting the announcement failed", "err", err)
		}

		// Home Assistant uses this to open a conversation without a wake word. An announcement that
		// was cut off was cut off on purpose, so nothing follows it.
		if a.StartConversation && !claim.Stopped() {
			t.Start(0)
		}
	})
}

// play fetches audio and queues it. Home Assistant serves it converted to media.Formats. Waiting
// for it to be heard is the driver's, not ours: this returns as soon as it has been handed over.
func (t *Conversation) play(ctx context.Context, url string, started func()) error {
	samples, err := pcm.Fetch(ctx, url)
	if err != nil {
		return err
	}
	slog.Debug("playing announcement", "samples", len(samples))

	if started != nil {
		started()
	}
	t.speaker.PlayVoice(samples)
	t.speaker.PlayVoice(make([]int16, 16000*pcm.Tail/1000))
	return nil
}
