package call

import (
	"context"
	"log/slog"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
	camerasession "github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

// Select chooses the saved stream, falling back to main when no substream is enabled.
func (c *Camera) Select(stream config.CallStream) (int, Size, bool) {
	sizes := c.sessions.Sizes()
	if len(sizes) == 0 {
		return 0, Size{}, false
	}
	if stream == config.CallSub && len(sizes) > 1 {
		return 1, sizes[1], true
	}
	return 0, sizes[0], true
}

// Camera adapts the shared vendor session hub to RTC frames and local preview.
type Camera struct {
	sessions *camerasession.Hub
}

// NewCamera uses the existing camera owner; it never opens vendor camera nodes.
func NewCamera(sessions *camerasession.Hub) *Camera {
	return &Camera{sessions: sessions}
}

// Key asks the camera owner for a new keyframe.
func (c *Camera) Key() {
	if err := c.sessions.RequestKey(); err != nil {
		slog.Debug("call keyframe not requested", "err", err)
	}
}

// Frames supplies encoded frames and rejoins when the camera session changes.
// sized may be nil. Release cancels the stream and waits for its worker to exit.
func (c *Camera) Frames(at int, sized func(Size)) (<-chan rtc.Frame, func()) {
	out := make(chan rtc.Frame, 8)
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	safe.Go("call camera", func() {
		defer close(done)
		defer close(out)
		var size Size
		for ctx.Err() == nil {
			s, frames, err := c.sessions.Join(at)
			if err != nil {
				slog.Warn("call camera", "err", err)
			} else {
				func() {
					defer c.sessions.Leave(s, frames)
					if sizes := c.sessions.Sizes(); at >= 0 && at < len(sizes) && sizes[at] != size {
						size = sizes[at]
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
	frames <-chan helper.Frame,
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
