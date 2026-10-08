package call

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type entities struct {
	incoming *esphome.Switch
	auto     *esphome.Switch
	wake     *esphome.Switch
	stream   *esphome.Select
	video    *esphome.Switch
	state    *esphome.TextSensor
	peer     *esphome.TextSensor
}

func (c *Calls) build() {
	c.ha = entities{
		incoming: &esphome.Switch{
			Base: esphome.Base{
				ObjectID: "call_incoming",
				Name:     "Incoming calls",
				Icon:     "mdi:phone-incoming",
				Category: esphome.CategoryConfig,
			},
		},
		auto: &esphome.Switch{
			Base: esphome.Base{
				ObjectID: "call_auto_answer",
				Name:     "Auto-answer calls",
				Icon:     "mdi:phone-in-talk",
				Category: esphome.CategoryConfig,
			},
		},
		wake: &esphome.Switch{
			Base: esphome.Base{
				ObjectID: "call_pause_wake",
				Name:     "Pause wake word during calls",
				Icon:     "mdi:microphone-off",
				Category: esphome.CategoryConfig,
			},
		},
		video: &esphome.Switch{
			Base: esphome.Base{
				ObjectID: "call_auto_video",
				Name:     "Auto-answer with camera on",
				Icon:     "mdi:video",
				Category: esphome.CategoryConfig,
			},
		},
		stream: &esphome.Select{
			Base: esphome.Base{
				ObjectID: "call_stream",
				Name:     "Call video stream",
				Icon:     "mdi:video-box",
				Category: esphome.CategoryConfig,
			},
			Options: labels(config.CallStreams()),
		},
		state: &esphome.TextSensor{
			Base: esphome.Base{ObjectID: "call_state", Name: "Call", Icon: "mdi:phone"},
		},
		peer: &esphome.TextSensor{
			Base: esphome.Base{ObjectID: "call_peer", Name: "Call with", Icon: "mdi:account-voice"},
		},
	}
	c.ha.incoming.OnCommand = c.SetIncoming
	c.ha.auto.OnCommand = c.SetAutoAnswer
	c.ha.wake.OnCommand = c.SetPauseWake
	c.ha.video.OnCommand = c.SetAutoVideo
	c.ha.stream.OnCommand = func(l string) {
		if v, ok := byLabel(config.CallStreams(), l); ok {
			c.SetStream(v)
		}
	}
}

func (c *Calls) Entities() []esphome.Entity {
	return []esphome.Entity{
		c.ha.incoming,
		c.ha.auto,
		c.ha.wake,
		c.ha.video,
		c.ha.stream,
		c.ha.state,
		c.ha.peer,
	}
}

func (c *Calls) Restore(cfg config.Call) {
	c.ha.incoming.Set(cfg.Incoming)
	c.ha.auto.Set(cfg.AutoAnswer)
	c.ha.wake.Set(cfg.PauseWake)
	c.ha.video.Set(cfg.AutoVideo)
	c.ha.stream.Set(cfg.Stream.Label())
	c.publish(nil)
}

func (c *Calls) publish(s *session) {
	if s == nil {
		c.ha.state.Set(Idle.String())
		c.ha.peer.Set("")
		return
	}
	c.mu.Lock()
	state, peer := s.State, s.Peer.Name
	c.mu.Unlock()
	c.ha.state.Set(state.String())
	c.ha.peer.Set(peer)
}

func (c *Calls) save(what string, err error) {
	if err != nil {
		slog.Error("saving a call setting failed", "setting", what, "err", err)
	}
	cfg := c.options.Settings.Read()
	c.ha.incoming.Set(cfg.Incoming)
	c.ha.auto.Set(cfg.AutoAnswer)
	c.ha.wake.Set(cfg.PauseWake)
	c.ha.video.Set(cfg.AutoVideo)
	c.ha.stream.Set(cfg.Stream.Label())
	c.options.Shell.Redraw()
}

func (c *Calls) SetIncoming(on bool) {
	c.save("incoming", c.options.Settings.Update(func(v *config.Call) { v.Incoming = on }))
}

func (c *Calls) SetAutoAnswer(on bool) {
	c.save("auto_answer", c.options.Settings.Update(func(v *config.Call) { v.AutoAnswer = on }))
}

func (c *Calls) SetPauseWake(on bool) {
	c.save("pause_wake", c.options.Settings.Update(func(v *config.Call) { v.PauseWake = on }))
}

func (c *Calls) SetAutoVideo(on bool) {
	c.save("auto_video", c.options.Settings.Update(func(v *config.Call) { v.AutoVideo = on }))
}

func (c *Calls) SetStream(v config.CallStream) {
	c.save("stream", c.options.Settings.Update(func(cfg *config.Call) { cfg.Stream = v }))
}

func labels(values []config.CallStream) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.Label()
	}
	return out
}
func byLabel(values []config.CallStream, label string) (config.CallStream, bool) {
	for _, v := range values {
		if v.Label() == label {
			return v, true
		}
	}
	return "", false
}
