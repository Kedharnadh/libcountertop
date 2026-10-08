package homeview

import (
	"errors"
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/homecontrol"
	"github.com/ygelfand/libcountertop/pkg/say"
	schema "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

func (view *Renderer) Body(w *gogui.Window) gogui.View {
	ha := view.options.Client
	device := map[string]any{"Name": view.options.Name()}
	on := view.options.Engine.Enabled()
	master := view.options.UI.Row(
		"enabled",
		"home",
		menu.Row{Label: say.T("home.enabled"), Kind: menu.Toggle, On: on},
		func(int) { view.options.Engine.SetEnabled(!on) },
		false,
	)
	switch {
	case !on:
		return Column([]gogui.View{master})
	case !ha.Connected():
		return Column(
			[]gogui.View{
				master,
				view.blocked(
					w,
					say.T("home.unconnected"),
					say.F("home.unconnected.hint", device),
					nil,
				),
			},
		)
	case ha.Access() == homeassistant.Unknown:
		return Column([]gogui.View{master, view.blocked(w, say.T("home.checking"), "", nil)})
	case ha.Access() == homeassistant.Refused:
		return Column(
			[]gogui.View{
				master,
				view.blocked(
					w,
					say.T("home.refused"),
					say.F("home.refused.hint", device),
					func(int) { ha.Probe() },
				),
			},
		)
	}

	views := []gogui.View{master}
	for i, s := range view.options.Engine.Selections() {
		views = append(views, view.options.UI.Row(fmt.Sprintf("sel-%d", i), "home",
			menu.Row{Label: s.Name(), Kind: menu.Chevron, Value: homecontrol.Summary(s.Pick())},
			func(int) { view.options.Shell.Push(view.options.Engine.SelectionPage(s)) }, false))
	}
	wc := view.options.Weather.Read()
	says := say.T("weather.off")
	if wc.Entity != "" {
		says = wc.Entity
	}
	views = append(views, view.options.UI.Row("weather", "home",
		menu.Row{Label: say.T("weather.title"), Kind: menu.Chevron, Value: says},
		func(int) { view.options.Shell.Push(view.options.WeatherPage()) }, false))
	views = append(views, Separator())
	views = append(views, view.options.UI.Row("refresh", "home",
		menu.Row{Label: say.T("home.refresh")},
		func(int) { view.options.Engine.Refresh() }, false))
	return Column(views)
}

func (view *Renderer) blocked(w *gogui.Window, title, hint string, retry func(int)) gogui.View {
	t := gogui.CurrentTheme().Cfg
	_, vh := w.WindowSize()
	side := min(t.TextStyleDef.Size*12, float32(vh)*0.2)
	big := t.TextStyleDef
	big.Size *= 1.3
	views := []gogui.View{
		gogui.Svg(
			gogui.SvgCfg{ID: "esphome", SvgData: view.options.Logo, Width: side, Height: side},
		),
		gogui.Label(title, big),
	}
	if hint != "" {
		views = append(
			views,
			gogui.Text(gogui.TextCfg{Text: hint, TextStyle: secondary(), Mode: gogui.TextModeWrap}),
		)
	}
	if retry != nil {
		views = append(
			views,
			view.options.UI.Row(
				"retry",
				"home",
				menu.Row{Glyph: gogui.IconSync, Label: say.T("home.check")},
				retry,
				true,
			),
		)
	}
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		HAlign:  gogui.HAlignCenter,
		Padding: gogui.PaddingLarge,
		Spacing: gogui.SpacingLarge,
		Content: views,
	})
}

func Separator() gogui.View {
	gap := gogui.CurrentTheme().Cfg.TextStyleDef.Size * 0.5
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NewPadding(gap, 0, gap, 0),
		Content: []gogui.View{gogui.Row(gogui.ContainerCfg{
			Height: 2,
			Sizing: gogui.FillFixed,
			Color:  gogui.CurrentTheme().Cfg.ColorBorder,
		})},
	})
}

func Column(views []gogui.View) gogui.View {
	return gogui.Column(
		gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingTight,
			Content: views,
		},
	)
}

func (view *Renderer) Picker(p *homecontrol.Picker) gogui.View {
	s := p.Snapshot()
	search := view.options.UI.Input(gogui.InputCfg{
		ID:            "home-search",
		Text:          s.Query,
		Placeholder:   say.T("home.search"),
		Sizing:        gogui.FillFit,
		OnTextChanged: func(q string, _ gogui.EventCtx) { p.SetQuery(q) },
	}, nil)

	var body []gogui.View
	switch {
	case s.Err != nil:
		body = []gogui.View{
			view.options.UI.Row(
				"failed",
				"picker",
				menu.Row{Label: say.T("home.failed"), Hint: view.failure(s.Err)},
				nil,
				false,
			),
			view.options.UI.Row(
				"retry",
				"picker",
				menu.Row{Glyph: gogui.IconSync, Label: say.T("home.retry")},
				func(int) { p.Retry() },
				true,
			),
		}
		return Column(append([]gogui.View{search}, body...))
	case s.Loading:
		return Column(
			[]gogui.View{
				search,
				view.options.UI.Row(
					"loading",
					"picker",
					menu.Row{Label: say.T("home.loading")},
					nil,
					false,
				),
			},
		)
	}

	var strip []gogui.View
	for _, it := range []struct {
		tab   homecontrol.Tab
		label string
	}{{homecontrol.TabLabels, say.T("home.tab.labels")}, {homecontrol.TabManual, say.T("home.tab.manual")}} {
		st := gogui.CurrentTheme().Cfg.TextStyleDef
		cfg := gogui.ContainerCfg{
			ID:      "home-tab-" + string(it.tab),
			Padding: gogui.PaddingMedium,
			VAlign:  gogui.VAlignMiddle,
		}
		view.options.UI.Toolkit.Tab(&cfg, &st, widgets.Chosen(s.Tab == it.tab))
		cfg.Content = []gogui.View{gogui.Label(it.label, st)}
		pick := it.tab
		strip = append(
			strip,
			view.options.UI.Toolkit.Pressable(
				gogui.Row,
				cfg,
				func(gogui.EventCtx) { p.SetTab(pick) },
			),
		)
	}
	shown := view.labelsTab(p, s)
	if s.Tab == homecontrol.TabManual {
		shown = view.manualTab(p, s)
	}
	tabs := gogui.Row(
		gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingSmall,
			Content: strip,
		},
	)
	return Column([]gogui.View{search, tabs, shown})
}

func (view *Renderer) failure(err error) string {
	device := map[string]any{"Name": view.options.Name()}
	switch {
	case errors.Is(err, homeassistant.ErrNotAllowed):
		return say.F("home.refused.hint", device)
	case errors.Is(err, homeassistant.ErrNotConnected):
		return say.F("home.unconnected.hint", device)
	}
	return ""
}

func (view *Renderer) labelsTab(p *homecontrol.Picker, s homecontrol.Snapshot) gogui.View {
	if len(s.Labels) == 0 {
		return Column(
			[]gogui.View{
				view.options.UI.Row(
					"empty",
					"labels",
					menu.Row{Label: say.T("home.empty")},
					nil,
					false,
				),
			},
		)
	}
	var views []gogui.View
	for i, l := range s.Labels {
		views = append(views, view.options.UI.Row(fmt.Sprintf("label-%d", i), "labels", menu.Row{
			Label: l.Name, Hint: say.F("home.label.count", map[string]any{"N": l.Count}),
			Kind: menu.Toggle, On: l.On, Chosen: l.On,
		}, func(int) { p.ToggleLabel(l.ID) }, false))
	}
	return Column(views)
}

func (view *Renderer) manualTab(p *homecontrol.Picker, s homecontrol.Snapshot) gogui.View {
	views := []gogui.View{view.options.UI.Row("all", "manual", menu.Row{
		Label: say.T("home.everything." + p.Sel.Key),
		Kind:  menu.Toggle, On: s.Pick.All, Chosen: s.Pick.All,
	}, func(int) { p.ToggleAll() }, false)}
	if len(s.Areas) == 0 {
		views = append(
			views,
			view.options.UI.Row(
				"empty",
				"manual",
				menu.Row{Label: say.T("home.empty")},
				nil,
				false,
			),
		)
	}
	for i, a := range s.Areas {
		views = append(views, view.areaHeader(p, s, i, a))
		if !a.Open {
			continue
		}
		for j, e := range a.Entities {
			name := e.Name
			if name == "" {
				name = e.ID
			}
			row := view.options.UI.Row(fmt.Sprintf("entity-%d-%d", i, j), "manual", menu.Row{
				Label: name, Kind: menu.Toggle, On: e.On, Chosen: e.Included, Dim: !e.Available(),
			}, func(int) { p.ToggleEntity(e.ID) }, false)
			views = append(views, gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NewPadding(0, 0, 0, reach()),
				Content: []gogui.View{row},
			}))
		}
	}
	return Column(views)
}

func (view *Renderer) areaHeader(
	p *homecontrol.Picker,
	s homecontrol.Snapshot,
	i int,
	a homecontrol.AreaRow,
) gogui.View {
	t := gogui.CurrentTheme().Cfg
	name := a.Name
	if a.ID == "" {
		name = say.T("home.noarea")
	}
	glyph := gogui.IconArrowRight
	if a.Open {
		glyph = gogui.IconArrowDown
	}
	id := fmt.Sprintf("area-%d", i)
	opener := view.options.UI.Toolkit.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:      id + "-open",
		Sizing:  gogui.FillFit,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		Content: []gogui.View{
			icon(glyph, t.ColorTextSecondary),
			gogui.Column(
				gogui.ContainerCfg{
					Sizing:  gogui.FillFit,
					Padding: gogui.NoPadding,
					Content: []gogui.View{
						gogui.Label(name, gogui.TextStyle{}),
						gogui.Label(
							say.F(
								"home.area.count",
								map[string]any{"N": a.Included, "Of": len(a.Entities)},
							),
							secondary(),
						),
					},
				},
			),
		},
	}, func(gogui.EventCtx) { p.Expand(a.ID) })

	content := []gogui.View{opener}
	if a.ID != "" {
		content = append(content, view.options.UI.Toolkit.Pressable(gogui.Row, gogui.ContainerCfg{
			ID:      id + "-switch",
			Padding: gogui.NoPadding,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{view.options.UI.Toolkit.Kit().Toggle(id+"-sw", a.On)},
		}, func(gogui.EventCtx) { p.ToggleArea(a.ID) }))
	}
	cfg := gogui.ContainerCfg{
		ID:      id,
		Sizing:  gogui.FillFit,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingMedium,
		Radius:  gogui.RadiusMedium,
		Content: content,
	}
	state := style.Rest
	switch {
	case a.On || s.Pick.All:
		state = style.Chosen
	case a.Included > 0:
		state = style.Partial
	}
	view.options.UI.Toolkit.ListRow(&cfg, state)
	return gogui.Row(cfg)
}

type Client interface {
	Connected() bool
	Access() homeassistant.Access
	Probe()
}
type Dependencies struct {
	Engine      *homecontrol.Engine
	Client      Client
	Name        func() string
	Weather     storage.Section[schema.Weather]
	WeatherPage func() shell.View
	Shell       *shell.Shell
	Logo        string
	UI          *widgets.UI
}
type Renderer struct{ options Dependencies }

func New(o Dependencies) *Renderer { return &Renderer{options: o} }
func reach() float32               { return style.Reach() }
func secondary() gogui.TextStyle {
	t := gogui.CurrentTheme().Cfg
	st := t.TextStyleDef
	st.Color = t.ColorTextSecondary
	return st
}
func icon(glyph string, c gogui.Color) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return gogui.Label(glyph, st)
}
