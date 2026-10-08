package turn

import (
	"context"
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/media/pcm"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

func (t *Conversation) announce(a esphome.Announce) {
	slog.Info("announce", "text", a.Text, "start_conversation", a.StartConversation)

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

		if a.StartConversation && !claim.Stopped() {
			t.Start(0)
		}
	})
}

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
