package tone

import (
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Player interface {
	Chime(schema.Stream, []Note)
	Sound(schema.Stream, float64, Sound)
}

// Sounds resolves built-in chimes and uploaded clips.
type Sounds interface {
	Sound(schema.Chime, schema.Chime) Sound
	Choices() []setting.Option
}

// FeedbackOptions selects the feedback controls used by the hardware.
type FeedbackOptions struct {
	Sounds Sounds
	// Mute exposes separate mute and unmute selections.
	Mute bool
	// Hold exposes the mute-button hold acknowledgement.
	Hold bool
}

type Feedback struct {
	*setting.Controls[schema.Feedback]
	settings storage.Section[schema.Feedback]
	player   Player
	sounds   Sounds
}

func NewFeedback(
	at storage.Section[schema.Feedback],
	player Player,
	deviceID uint32,
	options FeedbackOptions,
) *Feedback {
	var more func() []setting.Option
	if options.Sounds != nil {
		more = options.Sounds.Choices
	}
	f := &Feedback{settings: at, player: player, sounds: options.Sounds}
	f.Controls = &setting.Controls[schema.Feedback]{
		Table:  feedbackTable(more, options),
		Device: deviceID,
		Read:   at.Read,
		Save:   func(s setting.Setting[schema.Feedback], v string) error { return setting.Store(at, s, v) },
	}
	return f
}

func feedbackTable(
	more func() []setting.Option,
	options FeedbackOptions,
) *setting.Table[schema.Feedback] {
	const group setting.Group = "Feedback"
	row := func(name, id string, choices []schema.Chime, field func(*schema.Feedback) *schema.Chime) setting.Setting[schema.Feedback] {
		return setting.Pick(
			setting.Setting[schema.Feedback]{
				Name:  name,
				ID:    id,
				Group: group,
				Icon:  "mdi:music-note",
				More:  more,
			},
			choices,
			field,
		)
	}
	rows := []setting.Setting[schema.Feedback]{
		row(
			"chime",
			"device_chime",
			schema.Chimes(),
			func(c *schema.Feedback) *schema.Chime { return &c.Chime },
		),
		row(
			"failure",
			"",
			schema.AlertChimes(),
			func(c *schema.Feedback) *schema.Chime { return &c.Failure },
		),
		row(
			"cancel",
			"",
			append(schema.Chimes(), schema.ChimeCancel),
			func(c *schema.Feedback) *schema.Chime { return &c.Cancel },
		),
	}
	if options.Mute {
		rows = append(
			rows,
			row(
				"mute",
				"",
				append(schema.Chimes(), schema.ChimeMuted),
				func(c *schema.Feedback) *schema.Chime { return &c.Mute },
			),
			row(
				"unmute",
				"",
				append(schema.Chimes(), schema.ChimeUnmuted),
				func(c *schema.Feedback) *schema.Chime { return &c.Unmute },
			),
		)
	}
	if options.Hold {
		rows = append(
			rows,
			row(
				"hold",
				"",
				append(schema.Chimes(), schema.ChimeMuteHold),
				func(c *schema.Feedback) *schema.Chime { return &c.Hold },
			),
		)
	}
	return setting.NewTable(
		"feedback",
		[]setting.Group{group},
		rows,
		setting.Messages{Text: say.T, Missing: say.Missing},
	)
}

func (f *Feedback) Chime() schema.Chime { return f.settings.Read().Chime }
func (f *Feedback) SetChime(c schema.Chime) {
	if err := f.settings.Update(func(s *schema.Feedback) { s.Chime = c }); err != nil {
		slog.Error("saving a setting failed", "setting", "device_chime", "err", err)
	}
	f.Restore()
}
func (f *Feedback) Restore() { f.Publish() }
func (f *Feedback) Preview(s schema.Stream) {
	if s == schema.StreamMain {
		s = schema.StreamFeedback
	}
	f.selected(s, f.Chime(), schema.DefaultChime)
}

// Failure plays the selected failure sound unless device feedback is silenced.
func (f *Feedback) Failure() {
	f.selected(schema.StreamFeedback, f.settings.Read().Failure, schema.ChimeTrouble)
}

// Cancelled plays the selected cancellation sound unless device feedback is silenced.
func (f *Feedback) Cancelled() {
	f.selected(schema.StreamFeedback, f.settings.Read().Cancel, schema.ChimeCancel)
}

// Muted acknowledges microphone muting unless device feedback is silenced.
func (f *Feedback) Muted() {
	f.selected(schema.StreamFeedback, f.settings.Read().Mute, schema.ChimeMuted)
}

// Unmuted acknowledges microphone unmuting unless device feedback is silenced.
func (f *Feedback) Unmuted() {
	f.selected(schema.StreamFeedback, f.settings.Read().Unmute, schema.ChimeUnmuted)
}

// Hold acknowledges a held mute button, independently of the device chime switch.
func (f *Feedback) Hold() {
	f.playSelected(schema.StreamFeedback, f.settings.Read().Hold, schema.ChimeMuteHold)
}
func (f *Feedback) selected(s schema.Stream, c, fallback schema.Chime) {
	if !f.Chime().Silent() {
		f.playSelected(s, c, fallback)
	}
}
func (f *Feedback) playSelected(s schema.Stream, c, fallback schema.Chime) {
	if c == "" {
		c = fallback
	}
	snd := Sound{Notes: Wake(c)}
	if f.sounds != nil {
		snd = f.sounds.Sound(c, fallback)
	} else if _, custom := c.Custom(); custom {
		snd = Sound{Notes: Wake(fallback)}
	}
	if !snd.Empty() {
		f.player.Sound(s, Level, snd)
	}
}
func (f *Feedback) Sound(notes []Note) { f.Play(schema.StreamFeedback, notes) }
func (f *Feedback) Play(s schema.Stream, notes []Note) {
	if !f.Chime().Silent() {
		f.player.Chime(s, notes)
	}
}

// PreviewSetting plays a feedback selection independently of the master switch.
func (f *Feedback) PreviewSetting(name string) {
	row, ok := f.Table.Find(name)
	if !ok {
		return
	}
	at := f.Read()
	defaults := schema.DefaultFeedback()
	f.playSelected(
		schema.StreamFeedback,
		schema.Chime(row.Read(&at)),
		schema.Chime(row.Read(&defaults)),
	)
}
