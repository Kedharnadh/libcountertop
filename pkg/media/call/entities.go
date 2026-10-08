package call

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Setting = setting.Setting[config.Call]

const Group setting.Group = "Calls"

type entities struct {
	knobs *setting.Controls[config.Call]
	state *esphome.TextSensor
	peer  *esphome.TextSensor
}

func NewTable() *setting.Table[config.Call] {
	return setting.NewTable("call", []setting.Group{Group}, []Setting{
		setting.Flag(
			Setting{Name: "incoming", Group: Group, Icon: "mdi:phone-incoming"},
			func(c *config.Call) *bool { return &c.Incoming },
		),
		setting.Flag(
			Setting{Name: "auto_answer", Group: Group, Icon: "mdi:phone-in-talk"},
			func(c *config.Call) *bool { return &c.AutoAnswer },
		),
		setting.Flag(
			Setting{Name: "pause_wake", Group: Group, Icon: "mdi:microphone-off"},
			func(c *config.Call) *bool { return &c.PauseWake },
		),
		setting.Flag(
			Setting{Name: "auto_video", Group: Group, Icon: "mdi:video"},
			func(c *config.Call) *bool { return &c.AutoVideo },
		),
		setting.Pick(
			Setting{Name: "stream", Group: Group, Icon: "mdi:video-box"},
			config.CallStreams(),
			func(c *config.Call) *config.CallStream { return &c.Stream },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func (c *Calls) build() {
	c.ha = entities{
		knobs: &setting.Controls[config.Call]{
			Table: NewTable(),
			Read:  c.options.Settings.Read,
			Save:  c.store,
		},
		state: &esphome.TextSensor{
			Base: esphome.Base{ObjectID: "call_state", Name: "Call", Icon: "mdi:phone"},
		},
		peer: &esphome.TextSensor{
			Base: esphome.Base{ObjectID: "call_peer", Name: "Call with", Icon: "mdi:account-voice"},
		},
	}
}

func (c *Calls) Controls() *setting.Controls[config.Call] { return c.ha.knobs }

func (c *Calls) Entities() []esphome.Entity {
	return append(c.ha.knobs.Entities(), c.ha.state, c.ha.peer)
}

func (c *Calls) Restore(config.Call) {
	c.ha.knobs.Publish()
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

func (c *Calls) store(s Setting, v string) error {
	err := setting.Store(c.options.Settings, s, v)
	c.options.Shell.Redraw()
	return err
}

func (c *Calls) set(name, v string) {
	if err := c.ha.knobs.Change(name, v); err != nil {
		slog.Error("saving a call setting failed", "setting", name, "err", err)
		c.ha.knobs.Publish()
	}
}

func (c *Calls) SetIncoming(on bool) { c.set("incoming", setting.OnOff(on)) }

func (c *Calls) SetAutoAnswer(on bool) { c.set("auto_answer", setting.OnOff(on)) }

func (c *Calls) SetPauseWake(on bool) { c.set("pause_wake", setting.OnOff(on)) }

func (c *Calls) SetAutoVideo(on bool) { c.set("auto_video", setting.OnOff(on)) }

func (c *Calls) SetStream(v config.CallStream) { c.set("stream", string(v)) }
