package callstatus

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
)

type Size = session.Size
type Status struct {
	Muted     bool `json:"muted"`
	CameraOff bool `json:"camera_off"`
}
type Link interface {
	Send(string, any) error
	Muted() bool
	Sending() bool
}
type Options[S any] struct {
	Snapshot func(S) (Link, Size)
	Own      func(S, Size) (func(), bool)
	Remote   func(S, Size) (func(), bool)
	Status   func(S, Status) (func(), bool)
	Changed  func(S)
}
type Controller[S any] struct{ Options[S] }

func (c Controller[S]) Tell(s S) {
	l, _ := c.Snapshot(s)
	if l != nil {
		sent("status", l.Send("status", Status{Muted: l.Muted(), CameraOff: !l.Sending()}))
	}
}
func (c Controller[S]) TellSize(s S) {
	l, sz := c.Snapshot(s)
	if l != nil && sz.Width > 0 {
		sent("size", l.Send("size", sz))
	}
}
func sent(kind string, err error) {
	if err != nil && !errors.Is(err, rtc.ErrNotOpen) {
		slog.Debug("call "+kind+" not sent", "err", err)
	}
}
func (c Controller[S]) OwnSized(s S, sz Size) {
	after, changed := c.Own(s, sz)
	if !changed {
		return
	}
	slog.Info("call camera size", "width", sz.Width, "height", sz.Height)
	if after != nil {
		after()
	}
	c.TellSize(s)
	c.Changed(s)
}
func (c Controller[S]) Received(s S) func(rtc.Message) {
	return func(m rtc.Message) {
		var after func()
		var accepted bool
		switch m.Kind {
		case "status":
			var st Status
			if json.Unmarshal(m.Data, &st) != nil {
				return
			}
			after, accepted = c.Status(s, st)
		case "size":
			var sz Size
			if json.Unmarshal(m.Data, &sz) != nil || sz.Width <= 0 || sz.Height <= 0 {
				return
			}
			after, accepted = c.Remote(s, sz)
			if accepted {
				slog.Info("call remote size", "width", sz.Width, "height", sz.Height)
			}
		default:
			slog.Debug("call message ignored", "kind", m.Kind)
			return
		}
		if accepted {
			if after != nil {
				after()
			}
			c.Changed(s)
		}
	}
}
