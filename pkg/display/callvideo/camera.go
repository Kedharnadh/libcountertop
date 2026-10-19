package callvideo

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/camerafeed"
	camerasession "github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (c *Camera) Select(stream config.CallStream) (int, Size, bool) {
	sizes := c.sessions.Sizes()
	if len(sizes) == 0 {
		return 0, Size{}, false
	}
	if stream == config.CallSub && len(sizes) > 1 {
		return 1, Size(sizes[1]), true
	}
	return 0, Size(sizes[0]), true
}

type Camera struct {
	sessions *camerasession.Hub
}

func NewCamera(sessions *camerasession.Hub) *Camera {
	return &Camera{sessions: sessions}
}

func (c *Camera) Key() {
	if err := c.sessions.RequestKey(); err != nil {
		slog.Debug("call keyframe not requested", "err", err)
	}
}

func (c *Camera) Frames(at int, sized func(Size)) (<-chan rtc.Frame, func()) {
	out := make(chan rtc.Frame, 8)
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	safe.Go("call camera", func() {
		defer close(done)
		defer close(out)
		var size Size
		for ctx.Err() == nil {
			s, frames, err := c.sessions.JoinOrCard(at)
			switch {
			case errors.Is(err, camerasession.ErrMuted), errors.Is(err, camerasession.ErrShutter):
			case err != nil:
				slog.Warn("call camera", "err", err)
			default:
				func() {
					defer c.sessions.Leave(s, frames)
					if sizes := c.sessions.Sizes(); at >= 0 && at < len(sizes) &&
						Size(sizes[at]) != size {
						size = Size(sizes[at])
						if sized != nil {
							sized(size)
						}
					}
					pump(ctx, s, frames, out)
				}()
			}
			select {
			case <-ctx.Done():
			case <-time.After(rejoinWait):
			}
		}
	})
	return out, func() { stop(); <-done }
}

func pump(
	ctx context.Context,
	s *camerasession.Session,
	frames <-chan camerafeed.Frame,
	out chan<- rtc.Frame,
) {
	var config []byte
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			if f.Config {
				config = append([]byte(nil), f.Data...)
				continue
			}
			data := f.Data
			if f.Key && config != nil {
				data = append(append([]byte(nil), config...), f.Data...)
			}
			select {
			case out <- rtc.Frame{Data: data, PTS: f.PTS, Key: f.Key}:
			default:
			}
		}
	}
}

type camera struct {
	source *Camera
	at     int
	sized  func(Size)
}

func (c camera) Frames() (<-chan rtc.Frame, func()) { return c.source.Frames(c.at, c.sized) }
func (c camera) Key()                               { c.source.Key() }
