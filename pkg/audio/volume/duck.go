package volume

import (
	"math"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Ducking = setting.Controls[schema.Media]

const MediaGroup setting.Group = "Media"

func DuckTable() *setting.Table[schema.Media] {
	var t *setting.Table[schema.Media]
	t = setting.NewTable("media", []setting.Group{MediaGroup}, []setting.Setting[schema.Media]{
		{
			Name:  "duck",
			Group: MediaGroup,
			ID:    "media_duck_level",
			Icon:  "mdi:volume-medium",
			Kind:  setting.Number,
			Min:   ducking.MinimumDB,
			Max:   ducking.MaximumDB,
			Unit:  "dB",
			Read: func(m *schema.Media) string {
				return strconv.Itoa(int(math.Round(m.DuckDB)))
			},
			Write: func(m *schema.Media, v string) error {
				n, err := t.MustRow("duck").Number(v)
				if err != nil {
					return err
				}
				m.DuckDB = float64(n)
				return nil
			},
		},
		setting.Pick(
			setting.Setting[schema.Media]{
				Name:  "buttons",
				Group: MediaGroup,
				ID:    "volume_buttons",
				Icon:  "mdi:volume-vibrate",
			},
			schema.VolumeButtonModes(),
			func(m *schema.Media) *schema.VolumeButtons { return &m.Buttons },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
	return t
}

func NewDucking(at storage.Section[schema.Media], device uint32) *Ducking {
	return &Ducking{
		Table:  DuckTable(),
		Device: device,
		Read:   at.Read,
		Save: func(s setting.Setting[schema.Media], v string) error {
			return setting.Store(at, s, v)
		},
	}
}
