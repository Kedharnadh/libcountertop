package screencmd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/idleview"
	"github.com/ygelfand/libcountertop/pkg/display/screen"
	"github.com/ygelfand/libcountertop/pkg/display/settings"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/display/visualinput"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/weather"
	"github.com/ygelfand/libcountertop/pkg/runtime/control"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type State struct {
	Network schema.Network
	Clock   schema.Clock
	Idle    schema.Idle
	Screen  schema.Screen
	Visual  schema.Visual
	Weather schema.Weather
}

type Dependencies struct {
	Clock    *dashboard.Controls
	Idle     *idleview.Idle
	Visuals  *visualinput.Visuals
	Weather  *weather.Weather
	Screen   *screen.Screen
	Settings storage.Section[schema.Screen]
	Shell    *shell.Shell

	MediaIdle func(schema.Delay)
	Marks     func(bool)
	Verify    func(bool)
}

func at(
	name, field string,
	read func(State) string,
	write func(string) error,
) control.Setting[State] {
	return control.Setting[State]{Name: name, Field: field, Read: read, Write: write}
}

func Rows(d Dependencies) []control.Setting[State] {
	c, i, v, w := d.Clock, d.Idle, d.Visuals, d.Weather
	return []control.Setting[State]{
		at("network.verify", "Network.Verify",
			func(s State) string { return control.OnOff(s.Network.Verify) },
			control.Toggle(d.Verify)),

		at("clock.face", "Clock.Face",
			func(s State) string { return string(s.Clock.Face) },
			control.Choose(schema.Faces(), c.SetFace)),
		at("clock.position", "Clock.Position",
			func(s State) string { return string(s.Clock.Position) },
			control.Choose(schema.Positions(), c.SetPosition)),
		at("clock.size", "Clock.Size",
			func(s State) string { return string(s.Clock.Size) },
			control.Choose(schema.Sizes(), c.SetSize)),
		at("clock.color", "Clock.Ink",
			func(s State) string { return string(s.Clock.Ink) },
			control.Choose(schema.Inks(), c.SetInk)),
		at("clock.date", "Clock.Date",
			func(s State) string { return control.OnOff(s.Clock.Date) },
			control.Toggle(c.SetDate)),
		at("clock.hours", "Screen.Hours",
			func(s State) string { return string(s.Screen.Hours) },
			control.Choose(schema.HourFormats(), c.SetHours)),

		at("idle.after", "Idle.After",
			func(s State) string { return string(s.Idle.After) },
			control.Choose(schema.Delays(), i.SetAfter)),
		at("idle.media", "Idle.Media",
			func(s State) string { return string(s.Idle.Media) },
			control.Choose(schema.MediaDelays(), d.MediaIdle)),
		at("idle.face", "Idle.Face",
			func(s State) string { return string(s.Idle.Face) },
			control.Choose(schema.IdleFaces(), i.SetFace)),
		at("idle.position", "Idle.Position",
			func(s State) string { return string(s.Idle.Position) },
			control.Choose(schema.Positions(), i.SetPosition)),
		at("idle.align", "Idle.Align",
			func(s State) string { return string(s.Idle.Align) },
			control.Choose(schema.Aligns(), i.SetAlign)),
		at("idle.size", "Idle.Size",
			func(s State) string { return string(s.Idle.Size) },
			control.Choose(schema.Sizes(), i.SetSize)),
		at("idle.visual1", "Idle.First.Kind",
			func(s State) string { return control.OrNone(s.Idle.First.Kind) },
			idleKind(i, 0)),
		at("idle.visual1.source", "Idle.First.Source",
			func(s State) string { return string(s.Idle.First.Source) },
			control.Choose(schema.Sources(), func(src schema.Source) { i.SetSource(0, src) })),
		at("idle.visual2", "Idle.Second.Kind",
			func(s State) string { return control.OrNone(s.Idle.Second.Kind) },
			idleKind(i, 1)),
		at("idle.visual2.source", "Idle.Second.Source",
			func(s State) string { return string(s.Idle.Second.Source) },
			control.Choose(schema.Sources(), func(src schema.Source) { i.SetSource(1, src) })),

		at("screen.keyboard", "Screen.Keyboard",
			func(s State) string { return string(s.Screen.Keyboard) },
			control.Choose(schema.KeyboardSizes(), func(k schema.KeyboardSize) {
				settings.SetKeyboard(d.Settings, d.Shell, k)
			})),
		at("screen.language", "Screen.Language",
			func(State) string { return say.Chosen() },
			func(s string) error { return language(d, s) }),
		at("screen.theme", "Screen.Theme",
			func(s State) string { return s.Screen.Theme },
			func(s string) error { return paint(d, s) }),
		at("screen.style", "Screen.Style",
			func(s State) string { return d.Screen.Table().Row("style").Read(&s.Screen) },
			func(val string) error { return d.Screen.Set("style", val) }),
		at("screen.size", "Screen.Size",
			func(s State) string { return d.Screen.Table().Row("size").Read(&s.Screen) },
			func(val string) error { return d.Screen.Set("size", val) }),
		at("screen.marks", "Screen.Marks",
			func(s State) string { return control.OnOff(s.Screen.Marks) },
			control.Toggle(d.Marks)),
		at("screen.visual", "Visual.Kind",
			func(s State) string { return s.Visual.Kind },
			control.Choose(visual.Built(), v.SetKind)),
		at("screen.visual.fps", "Visual.MaxFPS",
			func(s State) string { return strconv.Itoa(s.Visual.MaxFPS) },
			func(s string) error { return fpsStep(v, s) }),
		at("screen.visual.seed", "Visual.Seed",
			func(s State) string { return strconv.Itoa(s.Visual.Seed) },
			control.Number(0, visual.SeedMost, v.SetSeed)),
		at("screen.visual.label", "Visual.Label",
			func(s State) string { return control.OrNone(s.Visual.Label) },
			control.Words(v.SetLabel)),

		at("weather.entity", "Weather.Entity",
			func(s State) string { return control.OrNone(s.Weather.Entity) },
			control.Words(w.SetEntity)),
		at("weather.look", "Weather.Look",
			func(s State) string { return string(s.Weather.Look) },
			control.Choose(schema.WeatherLooks(), w.SetLook)),
		at("weather.dashboard", "Weather.Dashboard",
			func(s State) string { return control.OnOff(s.Weather.Dashboard) },
			control.Toggle(w.SetDashboard)),
		at("weather.idle", "Weather.Idle",
			func(s State) string { return control.OnOff(s.Weather.Idle) },
			control.Toggle(w.SetIdle)),
		at("weather.animate", "Weather.Animate",
			func(s State) string { return control.OnOff(s.Weather.Animate) },
			control.Toggle(w.SetAnimate)),
		at("weather.themed", "Weather.Themed",
			func(s State) string { return control.OnOff(s.Weather.Themed) },
			control.Toggle(w.SetThemed)),
		at("weather.position", "Weather.Position",
			func(s State) string { return string(s.Weather.Position) },
			control.Choose(schema.Positions(), w.SetPosition)),
		at("weather.align", "Weather.Align",
			func(s State) string { return string(s.Weather.Align) },
			control.Choose(schema.Aligns(), w.SetAlign)),
		at("weather.size", "Weather.Size",
			func(s State) string { return string(s.Weather.Size) },
			control.Choose(schema.Sizes(), w.SetSize)),
	}
}

func idleKind(i *idleview.Idle, slot int) func(string) error {
	return func(s string) error {
		if strings.EqualFold(s, "none") || s == "" {
			i.SetKind(slot, "")
			return nil
		}
		for _, k := range visual.Built() {
			if strings.EqualFold(string(k), s) || strings.EqualFold(k.Label(), s) {
				i.SetKind(slot, string(k))
				return nil
			}
		}
		return fmt.Errorf(
			"want none or one of %s",
			strings.Join(control.Labels(visual.Built()), ", "),
		)
	}
}

func fpsStep(v *visualinput.Visuals, s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || !slices.Contains(schema.MaxFPSSteps, n) {
		return fmt.Errorf("want one of %v", schema.MaxFPSSteps)
	}
	v.SetMaxFPS(n)
	return nil
}

func language(d Dependencies, s string) error {
	for _, tag := range say.Languages() {
		if strings.EqualFold(tag, s) || strings.EqualFold(say.Name(tag), s) {
			settings.SetLanguage(d.Settings, d.Shell, tag)
			return nil
		}
	}
	return fmt.Errorf("want one of %s", strings.Join(say.Languages(), ", "))
}

func paint(d Dependencies, s string) error {
	names := []string{style.ThemeDefault}
	for _, t := range theme.All {
		names = append(names, t.Name)
	}
	for _, name := range names {
		if !strings.EqualFold(name, s) {
			continue
		}
		if err := schema.NewScreenWriter(d.Settings).Theme(name); err != nil {
			return err
		}
		d.Screen.Use(name)
		return nil
	}
	return fmt.Errorf("want one of %s", strings.Join(names[1:], ", "))
}
