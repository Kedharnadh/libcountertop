package widgets

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func (v *View) topBar(
	w *gogui.Window,
	at ui.Rect,
	l tessera.Layout,
	page int,
	strip bool,
	f frame,
) gogui.View {
	p := l.Pages[page]
	title := p.Title
	if title == "" {
		title = l.Title
	}
	if !f.fresh {
		title = say.T("tessera.lapsed")
	}
	band := float32(at.H)
	slot := band * 0.7
	reach := style.Reach()
	iconGap := reach * 0.12
	itemGap := reach * 0.4
	text := gogui.CurrentTheme().Cfg.TextStyleDef
	width := float32(at.W)

	key := slot
	mark, source := v.mark(f)
	if mark != nil {
		b := mark.Bounds()
		key = float32(math.Round(float64(slot) * float64(b.Dx()) / float64(b.Dy())))
	}
	var left []gogui.View
	switch {
	case p.Excluded && !strip:
		left = append(
			left,
			v.barKey("tessera-back", glyphView(0xF0141, slot), key, band, func(w *gogui.Window) {
				v.move(w, func() { v.back(l) })
			}),
		)
	case p.HomeControl:
		home := glyphView(0xF02DC, slot)
		if mark != nil {
			home = widgets.Picture(source, int(key), int(slot))
		}
		left = append(left, v.barKey("tessera-home", home, key, band, func(w *gogui.Window) {
			v.move(w, func() { v.home(l) })
		}))
	}
	if len(left) > 0 {
		left = append(left, gap(itemGap*0.6))
		width -= key + itemGap*0.6
	}
	now := time.Now()
	symbol := gogui.CurrentTheme().TextStyleIconMedium
	symbol.Family = icons.Family
	var items []gogui.View
	for _, it := range p.Bar {
		var parts []gogui.View
		if r := icons.Codepoint(it.Icon); r != 0 {
			st := symbol
			if c, ok := hexColor(it.Color); ok {
				st.Color = c
			}
			parts = append(parts, gogui.Label(icons.Glyph(r), st))
			width -= w.TextWidth(icons.Glyph(r), st)
		}
		if s := barText(it, l.Clock24, now); s != "" {
			if len(parts) > 0 {
				parts = append(parts, gap(iconGap))
				width -= iconGap
			}
			parts = append(parts, gogui.Label(s, text))
			width -= w.TextWidth(s, text)
		}
		if it.Kind == "clock" || it.Kind == "analog" || it.Kind == "date" || it.Kind == "ago" {
			v.tick(w, now, time.Minute)
		}
		if len(parts) == 0 {
			continue
		}
		items = append(items, gap(itemGap))
		width -= itemGap
		items = append(items, parts...)
	}
	head := text
	head.Size *= 1.3
	content := append(left, gogui.Label(widgets.Ellipsize(w, title, head, max(1, width)), head))
	content = append(
		content,
		gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding}),
	)
	content = append(content, items...)
	return widgets.Placed(at, gogui.Row(gogui.ContainerCfg{
		Width:   float32(at.W),
		Height:  band,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Spacing: gogui.NoSpacing,
		VAlign:  gogui.VAlignMiddle,
		Content: content,
	}))
}

func gap(width float32) gogui.View {
	return gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Height:  1,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
	})
}

func glyphView(r rune, size float32) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Family = icons.Family
	st.Size = size
	st.Color = gogui.CurrentTheme().Cfg.TextStyleDef.Color
	return gogui.Label(icons.Glyph(r), st)
}

func (v *View) mark(f frame) (image.Image, string) {
	if v.o.Logo == nil {
		return nil, ""
	}
	img := v.o.Logo.On(f.pal.Background)
	if img == nil || img.Bounds().Dy() == 0 {
		return nil, ""
	}
	return img, widgets.ImageSource(fmt.Sprintf("tessera-logo/%p", img), img)
}

func (v *View) barKey(
	id string,
	content gogui.View,
	width, band float32,
	tap func(*gogui.Window),
) gogui.View {
	return v.o.UI.Presses.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:      id,
		Width:   width,
		Height:  band,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{content},
	}, func(e gogui.EventCtx) {
		tap(e.Window)
		e.Window.InvalidateLayout()
	})
}

func (v *View) bottomStrip(at ui.Rect, l tessera.Layout, page int) gogui.View {
	if l.Pages[page].Excluded {
		return widgets.Placed(at, gogui.Row(gogui.ContainerCfg{
			Width:   float32(at.W),
			Height:  float32(at.H),
			Sizing:  gogui.FixedFixed,
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Padding: gogui.NoPadding,
			Content: []gogui.View{
				v.key(
					"tessera-back",
					gogui.IconArrowLeft,
					func(w *gogui.Window) { v.move(w, func() { v.back(l) }) },
				),
			},
		}))
	}
	return v.dots(at, pages(l), page)
}

func (v *View) key(id, glyph string, tap func(*gogui.Window)) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	return v.o.UI.Toolkit.IconKey(gogui.ContainerCfg{
		ID:      id,
		Width:   style.Reach(),
		Height:  style.Reach(),
		Sizing:  gogui.FixedFixed,
		Radius:  gogui.RadiusLarge,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
	}, glyph, st, style.Partial, func(e gogui.EventCtx) {
		tap(e.Window)
		e.Window.InvalidateLayout()
	})
}

func (v *View) tick(w *gogui.Window, now time.Time, every time.Duration) {
	at := now.Truncate(every).Add(every)
	v.mu.Lock()
	if !v.due.IsZero() && v.due.After(now) && !at.Before(v.due) {
		v.mu.Unlock()
		return
	}
	v.due = at
	v.mu.Unlock()
	time.AfterFunc(at.Sub(now), func() {
		v.mu.Lock()
		if v.due.Equal(at) {
			v.due = time.Time{}
		}
		v.mu.Unlock()
		w.QueueCommand(func(w *gogui.Window) { w.InvalidateLayout() })
	})
}

func barText(it tessera.BarItem, clock24 bool, now time.Time) string {
	switch it.Kind {
	case "clock", "analog":
		if clock24 {
			return now.Format("15:04")
		}
		return now.Format("3:04 PM")
	case "date":
		return now.Format("Mon 2 Jan")
	case "text":
		return it.Text
	case "ago":
		return ago(time.Unix(it.Epoch, 0), now)
	}
	return ""
}

func ago(at, now time.Time) string {
	d := now.Sub(at)
	past := d >= 0
	if !past {
		d = -d
	}
	var span string
	switch {
	case d < time.Minute:
		return say.T("tessera.now")
	case d < time.Hour:
		span = fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 24*time.Hour:
		span = fmt.Sprintf("%d h", int(d.Hours()))
	default:
		span = fmt.Sprintf("%d d", int(d.Hours()/24))
	}
	if past {
		return say.F("tessera.ago", map[string]any{"T": span})
	}
	return say.F("tessera.in", map[string]any{"T": span})
}

func hexColor(s string) (gogui.Color, bool) {
	if len(s) != 6 {
		return gogui.Color{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return gogui.Color{}, false
	}
	return gogui.RGB(uint8(v>>16), uint8(v>>8), uint8(v)), true
}
