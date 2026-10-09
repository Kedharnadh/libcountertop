package tone

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Player interface {
	Chime(schema.Stream, []Note)
}

type Feedback struct {
	settings storage.Section[schema.Feedback]
	player   Player
	choice   *esphome.Select
}

func NewFeedback(
	settings storage.Section[schema.Feedback],
	player Player,
	deviceID uint32,
) *Feedback {
	f := &Feedback{
		settings: settings,
		player:   player,
		choice: &esphome.Select{
			Base: esphome.Base{
				ObjectID: "device_chime",
				Name:     "Device chime",
				Icon:     "mdi:bell-ring-outline",
				Category: esphome.CategoryConfig,
				DeviceID: deviceID,
			},
		},
	}
	for _, c := range schema.Chimes() {
		f.choice.Options = append(f.choice.Options, c.Label())
	}
	f.choice.OnCommand = func(label string) {
		for _, c := range schema.Chimes() {
			if c.Label() == label {
				f.SetChime(c)
				return
			}
		}
	}
	return f
}
func (f *Feedback) Entities() []esphome.Entity { return []esphome.Entity{f.choice} }
func (f *Feedback) Chime() schema.Chime        { return f.settings.Read().Chime }
func (f *Feedback) SetChime(c schema.Chime) {
	if err := f.settings.Update(func(s *schema.Feedback) { s.Chime = c }); err != nil {
		slog.Error("saving a setting failed", "setting", "device_chime", "err", err)
	}
	f.Restore()
}
func (f *Feedback) Restore() { f.choice.Set(f.Chime().Label()) }
func (f *Feedback) Preview(s schema.Stream) {
	if s == schema.StreamMain {
		s = schema.StreamFeedback
	}
	f.Play(s, Wake(f.Chime()))
}
func (f *Feedback) Sound(notes []Note) { f.Play(schema.StreamFeedback, notes) }
func (f *Feedback) Play(s schema.Stream, notes []Note) {
	if !f.Chime().Silent() {
		f.player.Chime(s, notes)
	}
}
