package visualinput

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Setup struct {
	Visual config.Visual
	Lift   int
}

type (
	Setting = setting.Setting[Setup]
	Table   = setting.Table[Setup]
)

const Group setting.Group = "Visual"

func NewTable() *Table {
	return setting.NewTable("visual", []setting.Group{Group}, []Setting{
		setting.Pick(Setting{Name: "kind", Group: Group, ID: "visual", Icon: "mdi:waveform"},
			visual.Built(), func(s *Setup) *visual.Kind { return (*visual.Kind)(&s.Visual.Kind) }),
		fpsRow(Setting{Name: "fps", Group: Group, Icon: "mdi:speedometer"}),
		setting.Count(
			Setting{Name: "seed", Group: Group, Icon: "mdi:dice-multiple", Max: visual.SeedMost},
			func(s *Setup) *int { return &s.Visual.Seed },
		),
		setting.Words(Setting{Name: "label", Group: Group, Icon: "mdi:format-text"}, LabelMost,
			func(s *Setup) *string { return &s.Visual.Label }),
		setting.Count(Setting{
			Name: "lift", Group: Group, ID: "visualizer_lift", Icon: "mdi:equalizer",
			Max: LiftMax, Unit: "dB",
		}, func(s *Setup) *int { return &s.Lift }),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func fpsRow(s Setting) Setting {
	s.Kind = setting.Choice
	for _, n := range config.MaxFPSSteps {
		v := strconv.Itoa(n)
		s.Options = append(s.Options, setting.Option{Value: v, Label: v})
	}
	s.Read = func(c *Setup) string { return strconv.Itoa(c.Visual.MaxFPS) }
	s.Write = func(c *Setup, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil || !slices.Contains(config.MaxFPSSteps, n) {
			return fmt.Errorf("%s takes one of %v, not %q", s.Name, config.MaxFPSSteps, v)
		}
		c.Visual.MaxFPS = n
		return nil
	}
	return s
}
