package screens

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/idleview"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/display/visualview"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/homecontrol"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (a *App) screenFor(v shell.View) *Screen {
	switch t := v.(type) {
	case *idleview.View:
		return a.idleScreen(t)
	case *visualview.View:
		return a.visualScreen(t)
	case *videopage.Page:
		return a.videoScreen(t)
	case *shell.Page:
		return a.pageScreen(t)
	case *homecontrol.Home:
		return a.homeScreen(t)
	case *homecontrol.Picker:
		return a.pickerScreen(t)
	}
	for _, f := range a.o.Views {
		if s := f(v); s != nil {
			return s
		}
	}
	switch v {
	case a.o.PlayerPage():
		return &Screen{Title: say.T("player.now"), View: v, Fixed: true, Build: a.player.Body}
	case a.o.Volume.Card():
		return &Screen{View: v, Build: a.volumeCard}
	case a.o.Drawer:
		return &Screen{View: v, Build: func(w *gogui.Window) gogui.View { return a.rail(w, v) }}
	}
	return nil
}

func (a *App) pageScreen(p *shell.Page) *Screen {
	return &Screen{
		Title: p.Title,
		View:  p,
		Build: func(w *gogui.Window) gogui.View {
			return widgets.PageBody(w, p, widgets.PageRenderer{
				Row: a.ui.Row, Tile: a.tileView, Preview: a.livePreview, Beside: widgets.Beside,
			})
		},
	}
}

func (a *App) videoScreen(p *videopage.Page) *Screen {
	return &Screen{
		View:  p,
		Fixed: true,
		Build: func(w *gogui.Window) gogui.View { return a.player.Video(w, p) },
	}
}

func (a *App) homeScreen(h *homecontrol.Home) *Screen {
	return &Screen{Title: say.T("home.title"), View: h, Build: a.home.Body}
}

func (a *App) pickerScreen(p *homecontrol.Picker) *Screen {
	return &Screen{
		Title: p.Sel.Name(),
		View:  p,
		Build: func(*gogui.Window) gogui.View { return a.home.Picker(p) },
	}
}

func (a *App) rail(w *gogui.Window, v shell.View) gogui.View {
	var entries []widgets.RailEntry
	for _, e := range a.o.Drawer.Entries() {
		glyph := ""
		if e.Glyph != nil {
			glyph = e.Glyph()
		}
		entries = append(entries, widgets.RailEntry{Label: e.Label(), Glyph: glyph, Open: e.Open})
	}
	return a.ui.Toolkit.Rail(
		w,
		widgets.RailOptions{
			Entries: entries,
			Edge:    a.o.Settings().Screen.Drawer,
			Close:   func() { a.o.Shell.Remove(v) },
		},
	)
}

func (a *App) volumeCard(w *gogui.Window) gogui.View {
	vol := a.o.Volume
	stream, open := vol.Picked()
	var choices []widgets.VolumeChoice
	for _, s := range a.o.Streams {
		choices = append(
			choices,
			widgets.VolumeChoice{Key: string(s), Label: s.Label(), Level: vol.Level(s)},
		)
	}
	return a.ui.Toolkit.VolumeCard(w, widgets.VolumeOptions{
		Current: widgets.VolumeChoice{
			Key:   string(stream),
			Label: stream.Label(),
			Level: vol.Level(stream),
		},
		Choices: choices,
		Open:    open,
		Left:    a.o.Settings().Screen.Volume == schema.EdgeLeft,
		Set:     func(s string, v int) { vol.Set(schema.Stream(s), v) },
		Linger:  vol.Linger,
		Expand:  vol.Expand,
		Dismiss: vol.Dismiss,
		Grip:    a.ui.Grip,
	})
}

func (a *App) keyboard(w *gogui.Window) gogui.View {
	a.mu.Lock()
	r := a.r
	a.mu.Unlock()
	if r == nil {
		return nil
	}
	at, view := a.ui.Editor.Keyboard(w, widgets.KeyboardOptions{
		Size:    widgets.KeyboardSize(a.o.Settings().Screen.Keyboard),
		Palette: a.ui.Palette(),
		Kit:     a.ui.Kit(),
		Cancel:  say.T("keyboard.cancel"),
		Save:    say.T("keyboard.save"),
		Type:    r.Type,
		Press:   r.Press,
	})
	return widgets.Placed(at, view)
}

func (a *App) upgradeCard(w *gogui.Window) gogui.View {
	up := a.o.Upgrade()
	if !up.Active {
		return nil
	}
	pal := a.ui.Palette()
	img, key := a.o.Logo(), "logo/light"
	if theme.Dark(pal.Background) {
		img, key = a.o.Night(), "logo/night"
	}
	return widgets.UpgradeCard(w, up, pal, clockview.ImageSource(key, img))
}

func iconStyle(c gogui.Color) gogui.TextStyle {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return st
}

func icon(glyph string, c gogui.Color) gogui.View { return gogui.Label(glyph, iconStyle(c)) }

func secondary() gogui.TextStyle {
	t := gogui.CurrentTheme().Cfg
	st := t.TextStyleDef
	st.Color = t.ColorTextSecondary
	return st
}

func reach() float32 { return style.Reach() }

func (a *App) livePreview(w *gogui.Window, watch func(ui.Rect) bool) gogui.View {
	return a.preview.View(w, say.T("camera.starting"), watch)
}

func (a *App) tileView(w *gogui.Window, id, page string, c menu.Cell, tap func(int)) gogui.View {
	t := gogui.CurrentTheme().Cfg
	tw := int(t.SizeTextMedium * 8)
	th := int(float32(tw) * 0.72)
	content := []gogui.View{}
	build, native := clockview.Of(c.Face)
	switch {
	case c.Weather != "":
		content = append(content, a.weatherSample(w, c.Weather, float32(tw), float32(th)))
	case c.Style != "":
		content = append(content, widgets.StyleSample(c.Style, float32(tw), float32(th)))
	case c.Palette != nil:
		content = append(content, widgets.Swatch(*c.Palette, float32(tw), float32(th)))
	case native:
		content = append(content, a.faceTile(w, build, tw, th))
	case c.Paint != nil:
		content = append(
			content,
			widgets.Picture(
				a.painted(
					"tile/"+page+"/"+c.Label+"/"+time.Now().Format("15:04"),
					tw,
					th,
					a.ui.Palette().Surface,
					c.Paint,
				),
				tw,
				th,
			),
		)
	}
	return a.ui.Toolkit.Tile(id, c.Label, c.Chosen, content, tap)
}

func (a *App) painted(key string, w, h int, fill theme.Color, draw widgets.Paint) string {
	return widgets.Painted(key, w, h, fill, a.ui.Palette(), draw)
}

func (a *App) faceTile(w *gogui.Window, build clockview.View, tw, th int) gogui.View {
	cfg := a.o.Settings()
	pal := a.ui.Palette()
	r := face.Read(time.Now(), cfg.Screen.Hours == schema.TwentyFourHour).Undated()
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(tw),
		Height:  float32(th),
		Sizing:  gogui.FixedFixed,
		Color:   color(pal.Surface),
		Radius:  gogui.RadiusMedium,
		Padding: gogui.NoPadding,
		Clip:    true,
		Content: []gogui.View{build(w, ui.Rect{W: tw, H: th}, r, cfg.Clock.Ink.Over(pal))},
	})
}
