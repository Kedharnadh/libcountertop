package mix

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"

	"github.com/ygelfand/libcountertop/pkg/audio/alsa"
)

func Send(ctx context.Context, to io.Writer, buf []byte, underruns *atomic.Uint64) error {
	for {
		_, err := to.Write(buf)
		if err == nil {
			return nil
		}
		if !errors.Is(err, alsa.ErrUnderrun) {
			return err
		}
		if n := underruns.Add(1); n == 1 || n%100 == 0 {
			slog.Warn("playback underrun", "times", n)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}
