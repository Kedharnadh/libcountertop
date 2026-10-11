package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

type EntityTile struct {
	ID            string
	Width, Height float32
	Glyph, Family string
	Name, Status  string
	Value, Unit   string
	On, Available bool
	Accent        *gogui.Color
	Ground        *gogui.Color
	Corner        gogui.View
	More          []gogui.View
	Control       func(width float32) gogui.View
	ControlWidth  float32
	Picture       func(width, height float32) gogui.View
	Body          func(width, height float32) gogui.View
	Tap, Hold     func(gogui.EventCtx)
	Pan           func(gogui.EventCtx)
	Window        *gogui.Window
}

func Tint(on bool, pal theme.Theme) (badgeFill, badgeInk, fill gogui.Color) {
	if on {
		return lit(Color(pal.Roles.Warm))
	}
	return gogui.RGB(pal.Background.R, pal.Background.G, pal.Background.B).WithOpacity(0.4),
		gogui.RGB(pal.Text.R, pal.Text.G, pal.Text.B),
		gogui.RGB(pal.Surface.R, pal.Surface.G, pal.Surface.B)
}

func lit(lamp gogui.Color) (badgeFill, badgeInk, fill gogui.Color) {
	return lamp, inkOn(lamp), lamp.WithOpacity(0.22)
}

func DomainGlyph(domain string) string {
	switch domain {
	case "light":
		return gogui.IconSunnyO
	case "scene", "script", "automation", "button", "input_button":
		return gogui.IconPlay
	}
	return gogui.IconPlug
}

func KeyFill(pal theme.Theme) gogui.Color {
	return gogui.RGB(pal.Background.R, pal.Background.G, pal.Background.B).WithOpacity(0.4)
}

func (u *UI) Entity(t EntityTile) gogui.View {
	pal := u.Palette()
	badgeFill, badgeInk, fill := Tint(t.On, pal)
	if t.On && t.Accent != nil {
		badgeFill, badgeInk, fill = lit(*t.Accent)
	}
	if t.Ground != nil {
		fill = *t.Ground
	}
	pad := style.Reach() * 0.25
	gap := pad * 0.6
	inner := t.Width - 2*pad
	side := style.Reach() * 1.1
	if t.Height > 0 {
		side = min(side, (t.Height-2*pad)*0.45)
	}
	var content []gogui.View
	switch {
	case t.Body != nil && t.Height > 0:
		content = []gogui.View{t.Body(inner, t.Height-2*pad)}
	case t.Picture != nil && t.Height > 0:
		content = t.picture(inner, t.Height-2*pad, gap)
	case t.Value != "":
		content = t.watch(side*0.6, inner, gap, badgeFill, badgeInk)
	case t.Height > 0 && t.Width >= 1.8*t.Height:
		content = t.row(side, inner, gap, badgeFill, badgeInk)
	case t.Control != nil && t.Height > 0 && !t.roomy(side, gap):
		content = t.compact(side*0.75, inner, gap, badgeFill, badgeInk)
	default:
		content = t.stack(side, inner, badgeFill, badgeInk)
	}
	sizing := gogui.FixedFit
	if t.Height > 0 {
		sizing = gogui.FixedFixed
	}
	cfg := gogui.ContainerCfg{
		ID:       t.ID,
		Width:    t.Width,
		Height:   t.Height,
		Sizing:   sizing,
		Padding:  gogui.PadAll(pad),
		Spacing:  gogui.SpacingPx(gap),
		Radius:   gogui.RadiusLarge,
		Color:    fill,
		Clip:     true,
		Disabled: !t.Available,
		Content:  content,
	}
	if t.Hold != nil || t.Pan != nil {
		hold, pan := t.Hold, t.Pan
		cfg.OnGesture = func(e gogui.EventCtx) {
			switch {
			case hold != nil && e.Event.GestureType == gogui.GestureLongPress &&
				e.Event.GesturePhase == gogui.GesturePhaseBegan:
				hold(e)
				e.Consume()
			case pan != nil && e.Event.GestureType == gogui.GesturePan:
				pan(e)
				e.Consume()
			}
		}
	}
	u.Kit().Tile(&cfg, Chosen(t.On))
	return u.Presses.Pressable(gogui.Column, cfg, t.Tap)
}

func (t EntityTile) badge(side float32, fill, ink gogui.Color) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = ink
	st.Size = side * 0.5
	if t.Family != "" {
		st.Family = t.Family
	}
	return gogui.Row(gogui.ContainerCfg{
		Width:   side,
		Height:  side,
		Sizing:  gogui.FixedFixed,
		Radius:  gogui.RadiusLarge,
		Color:   fill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Content: []gogui.View{gogui.Label(t.Glyph, st)},
	})
}

func (t EntityTile) text(s string, st gogui.TextStyle, width float32) gogui.View {
	if t.Window != nil {
		s = Ellipsize(t.Window, s, st, width)
	}
	return gogui.Label(s, st)
}

func (t EntityTile) lines(width float32) []gogui.View {
	out := []gogui.View{t.text(t.Name, gogui.CurrentTheme().Cfg.TextStyleDef, width)}
	if t.Status != "" {
		out = append(out, t.text(t.Status, Secondary(), width))
	}
	return out
}

func (t EntityTile) stack(side, inner float32, fill, ink gogui.Color) []gogui.View {
	top := []gogui.View{
		t.badge(side, fill, ink),
		gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding}),
	}
	if t.Corner != nil {
		top = append(top, t.Corner)
	}
	content := []gogui.View{gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NoPadding,
		VAlign:  gogui.VAlignMiddle,
		Content: top,
	})}
	content = append(content, t.lines(inner)...)
	content = append(content, t.More...)
	if t.Control != nil {
		if t.Height > 0 {
			content = append(content, gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFill,
				Padding: gogui.NoPadding,
			}))
		}
		content = append(content, t.Control(inner))
	}
	return content
}

func (t EntityTile) picture(width, height, gap float32) []gogui.View {
	lines := t.lines(width)
	line := gogui.CurrentTheme().Cfg.TextStyleDef.Size * 1.3 * float32(len(lines))
	return append([]gogui.View{gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Height:  max(1, height-line-gap),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{t.Picture(width, max(1, height-line-gap))},
	})}, lines...)
}

func (t EntityTile) roomy(side, gap float32) bool {
	line := gogui.CurrentTheme().Cfg.TextStyleDef.Size * 1.3
	need := side + gap + line + style.Reach() + 2*gap
	if t.Status != "" {
		need += line + gap
	}
	pad := style.Reach() * 0.25
	return need <= t.Height-2*pad
}

func (t EntityTile) compact(side, inner, gap float32, fill, ink gogui.Color) []gogui.View {
	words := max(1, inner-side-gap)
	return []gogui.View{
		gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap),
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{
				t.badge(side, fill, ink),
				gogui.Column(gogui.ContainerCfg{
					Width:   words,
					Sizing:  gogui.FixedFit,
					Padding: gogui.NoPadding,
					Content: t.lines(words),
				}),
			},
		}),
		gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding}),
		t.Control(inner),
	}
}

func (t EntityTile) row(side, inner, gap float32, fill, ink gogui.Color) []gogui.View {
	control := float32(0)
	var tail []gogui.View
	if t.Control != nil {
		control = inner * 0.5
		if t.ControlWidth > 0 {
			control = min(control, t.ControlWidth)
		}
		tail = []gogui.View{gogui.Row(gogui.ContainerCfg{
			Width:   control,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			HAlign:  gogui.HAlignRight,
			Content: []gogui.View{t.Control(control)},
		})}
	}
	words := max(1, inner-side-control-2*gap)
	body := []gogui.View{
		t.badge(side, fill, ink),
		gogui.Column(gogui.ContainerCfg{
			Width:   words,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Content: t.lines(words),
		}),
	}
	return []gogui.View{gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		VAlign:  gogui.VAlignMiddle,
		Content: append(body, tail...),
	})}
}

func (t EntityTile) watch(side, inner, gap float32, fill, ink gogui.Color) []gogui.View {
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2.6
	unit := Secondary()
	words := max(1, inner-side-gap)
	number := []gogui.View{gogui.Label(t.Value, big)}
	if t.Unit != "" {
		number = append(number, gogui.Label(t.Unit, unit))
	}
	return []gogui.View{
		gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap),
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{
				t.badge(side, fill, ink),
				t.text(t.Name, gogui.CurrentTheme().Cfg.TextStyleDef, words),
			},
		}),
		gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding}),
		gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap * 0.5),
			VAlign:  gogui.VAlignBottom,
			Content: number,
		}),
	}
}

func Ellipsize(w *gogui.Window, text string, st gogui.TextStyle, width float32) string {
	if w == nil || width <= 0 || w.TextWidth(text, st) <= width {
		return text
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if w.TextWidth(string(runes[:mid])+"…", st) <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(runes[:lo]) + "…"
}

func inkOn(c gogui.Color) gogui.Color {
	if 299*int(c.R)+587*int(c.G)+114*int(c.B) > 150000 {
		return gogui.RGB(40, 28, 0)
	}
	return gogui.RGB(255, 255, 255)
}
