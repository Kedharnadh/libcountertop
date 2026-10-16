package alarmview

import (
	"time"

	"github.com/go-gui-org/go-glyph"
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/assistant/alarm"
	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	mdi "github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Alarms interface {
	Current() (alarm.Alarm, bool)
	Dawn() alarm.Dawn
	Stop() bool
	Snooze() bool
}

type View struct {
	UI      *widgets.UI
	Alarms  Alarms
	Again   func(w *gogui.Window, after time.Duration)
	Clock24 func() bool
}

func (v *View) Ringing() bool {
	_, ok := v.Alarms.Current()
	return ok
}

func (v *View) Build(w *gogui.Window) gogui.View {
	al, ok := v.Alarms.Current()
	if !ok {
		if d := v.Alarms.Dawn(); d.Full {
			return v.overScene(w, d.Alarm)
		}
		return nil
	}
	if al.Light > 0 {
		return v.overScene(w, al)
	}
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	fw, fh := float32(vw), float32(vh)
	pal := v.UI.Palette()

	deeper := widgets.White
	if theme.Dark(pal.Accent) {
		deeper = widgets.Black
	}
	back := widgets.Pulse(w, v.Again, pal.Accent, deeper)
	ink := pal.Contrast(pal.Accent)

	label := v.clock(al)
	big := clockview.Lettering(glyph.TypefaceBold, 0, ink)
	big.Size = clockview.Fitted(w, label, big, fw*0.8, fh*0.38)

	bell := widgets.IconStyle(widgets.Color(ink))
	bell.Family, bell.Size = mdi.Family, big.Size*0.4
	middle := []gogui.View{
		gogui.Label(mdi.Glyph(mdi.Named("alarm")), bell),
		gogui.Label(label, big),
	}
	if al.Name != "" {
		st := t.TextStyleTitle
		st.Color, st.Align = widgets.Color(ink), gogui.TextAlignCenter
		middle = append(middle, gogui.Label(al.Name, st))
	}

	tall := style.Reach() * 1.6
	keys := []gogui.View{}
	if al.Snooze > 0 {
		keys = append(keys, v.key(w, "alarm-snooze",
			say.F("alarm.snooze_for", map[string]any{"N": int(al.Snooze / time.Minute)}),
			tall, fw*0.3, widgets.Color(ink).WithOpacity(0.18), widgets.Color(ink),
			func(gogui.EventCtx) { v.Alarms.Snooze() },
		))
	}
	keys = append(keys, v.key(w, "alarm-stop", say.T("alarm.stop"), tall, fw*0.3,
		widgets.Color(ink), widgets.Color(pal.Accent),
		func(gogui.EventCtx) { v.Alarms.Stop() },
	))

	return gogui.Column(widgets.Blocking(gogui.ContainerCfg{
		ID:      "alarm",
		Width:   fw,
		Height:  fh,
		Sizing:  gogui.FixedFixed,
		Color:   widgets.Color(back),
		Radius:  gogui.NoRadius,
		Padding: gogui.PaddingLarge,
		Spacing: gogui.SpacingLarge,
		Content: []gogui.View{
			gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFill,
				HAlign:  gogui.HAlignCenter,
				VAlign:  gogui.VAlignMiddle,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingMedium,
				Clip:    true,
				Content: middle,
			}),
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				HAlign:  gogui.HAlignCenter,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingLarge,
				Content: keys,
			}),
		},
	}))
}

func (v *View) overScene(w *gogui.Window, al alarm.Alarm) gogui.View {
	vw, vh := w.WindowSize()
	fw, fh := float32(vw), float32(vh)
	pal := v.UI.Palette()
	tall := style.Reach() * 1.6

	keys := []gogui.View{}
	if al.Snooze > 0 {
		keys = append(keys, v.key(w, "alarm-snooze",
			say.F("alarm.snooze_for", map[string]any{"N": int(al.Snooze / time.Minute)}),
			tall, fw*0.3, gogui.Black.WithOpacity(0.45), gogui.White,
			func(gogui.EventCtx) { v.Alarms.Snooze() },
		))
	}
	keys = append(keys, v.key(w, "alarm-stop", say.T("alarm.stop"), tall, fw*0.3,
		widgets.Color(pal.Accent), widgets.Color(pal.Contrast(pal.Accent)),
		func(gogui.EventCtx) { v.Alarms.Stop() },
	))

	bottom := []gogui.View{}
	if al.Name != "" {
		st := gogui.CurrentTheme().TextStyleTitle
		st.Color, st.Align = gogui.White, gogui.TextAlignCenter
		bottom = append(bottom, gogui.Row(gogui.ContainerCfg{
			Color:   gogui.Black.WithOpacity(0.35),
			Radius:  gogui.RadiusLarge,
			Padding: gogui.PaddingMedium,
			Content: []gogui.View{gogui.Label(al.Name, st)},
		}))
	}
	bottom = append(bottom, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		HAlign:  gogui.HAlignCenter,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingLarge,
		Content: keys,
	}))

	return gogui.Column(widgets.Blocking(gogui.ContainerCfg{
		ID:      "alarm",
		Width:   fw,
		Height:  fh,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.PaddingLarge,
		Spacing: gogui.SpacingMedium,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignBottom,
		Content: bottom,
	}))
}

func (v *View) key(
	w *gogui.Window,
	id, text string,
	tall, least float32,
	fill, ink gogui.Color,
	press func(gogui.EventCtx),
) gogui.View {
	st := gogui.CurrentTheme().TextStyleTitle
	st.Color = ink
	return v.UI.Presses.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:      id,
		Width:   max(widgets.KeyWidth(w, text, st, tall), least),
		Height:  tall,
		Sizing:  gogui.FixedFixed,
		Radius:  gogui.RadiusLarge,
		Color:   fill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Content: []gogui.View{gogui.Label(text, st)},
	}, press)
}

func (v *View) clock(al alarm.Alarm) string {
	if v.Clock24 != nil && v.Clock24() {
		return al.Clock()
	}
	return time.Date(2000, 1, 1, al.Hour, al.Minute, 0, 0, time.UTC).Format("3:04")
}
