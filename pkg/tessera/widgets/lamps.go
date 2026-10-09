package widgets

import (
	"strconv"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	subLamps   = "lamps"
	subEffects = "effects"
)

type sub struct {
	kind   string
	lamp   string
	pick   string
	page   int
	source int
}

func (v *View) aside() sub {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.sub
}

func (v *View) setAside(s sub) {
	v.mu.Lock()
	v.sub = s
	v.mu.Unlock()
}

func (v *View) backAside(*gogui.Window) {
	v.mu.Lock()
	defer v.mu.Unlock()
	switch {
	case v.sub.lamp != "":
		v.sub.lamp = ""
	case v.sub.pick != "":
		v.sub.pick, v.sub.page, v.sub.source = "", 0, 0
	default:
		v.sub = sub{}
	}
}

func (v *View) asideKeys(t tessera.Tile, width float32) []gogui.View {
	var keys []gogui.View
	var kinds [][3]string
	if effectsPage(t) {
		kinds = append(
			kinds,
			[3]string{"tessera-effects", say.T("tessera.light.effects"), subEffects},
		)
	}
	if len(t.Lamps()) > 0 {
		kinds = append(kinds, [3]string{"tessera-lamps", say.T("tessera.light.lamps"), subLamps})
	}
	if len(kinds) == 0 {
		return nil
	}
	gap := style.Reach() * 0.25
	each := (width - gap*float32(len(kinds)-1)) / float32(len(kinds))
	for _, k := range kinds {
		kind := k[2]
		keys = append(keys, v.wordKey(k[0], k[1], each, false,
			func(gogui.EventCtx) { v.setAside(sub{kind: kind}) }))
	}
	return []gogui.View{gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: keys,
	})}
}

func (v *View) lampLook(t tessera.Tile, fresh bool) kit.EntityTile {
	e := kit.EntityTile{On: active(t), Available: fresh && available(t)}
	if e.On {
		a := accent(t, v.o.UI.Palette())
		c := gogui.RGB(a.R, a.G, a.B)
		e.Accent = &c
	}
	return e
}

func lampOf(t tessera.Tile, entity string) (tessera.Tile, bool) {
	for _, l := range t.Lamps() {
		if l.Entity == entity {
			return l, true
		}
	}
	return tessera.Tile{}, false
}

func (v *View) lampsBody(
	w *gogui.Window,
	t tessera.Tile,
	fresh bool,
	width, room float32,
) []gogui.View {
	reach := style.Reach()
	s := v.aside()
	if l, ok := lampOf(t, s.lamp); ok {
		e := v.lampLook(l, fresh)
		return append(
			[]gogui.View{v.lampRow(w, l, e, width, false)},
			v.lightBody(w, l, e, width)...)
	}
	lamps := t.Lamps()
	per := max(1, int((room-reach*1.4)/(reach*1.35)))
	pages := (len(lamps) + per - 1) / per
	page := min(max(s.page, 0), max(pages-1, 0))
	var out []gogui.View
	for _, l := range lamps[page*per : min(len(lamps), (page+1)*per)] {
		out = append(out, v.lampRow(w, l, v.lampLook(l, fresh), width, true))
	}
	if pages > 1 {
		turn := func(to int) func(*gogui.Window) {
			return func(*gogui.Window) {
				v.mu.Lock()
				v.sub.page = to
				v.mu.Unlock()
			}
		}
		keys := []gogui.View{}
		if page > 0 {
			keys = append(keys, v.key("tessera-lamps-back", gogui.IconArrowLeft, turn(page-1)))
		}
		keys = append(keys, gogui.Label(
			strconv.Itoa(page+1)+" / "+strconv.Itoa(pages), secondary()))
		if page < pages-1 {
			keys = append(keys, v.key("tessera-lamps-next", gogui.IconArrowRight, turn(page+1)))
		}
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.3),
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: keys,
		}))
	}
	return out
}

func (v *View) lampRow(
	w *gogui.Window,
	l tessera.Tile,
	e kit.EntityTile,
	width float32,
	more bool,
) gogui.View {
	reach := style.Reach()
	id := "tessera-lamp-" + l.Entity
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	nameW := width * 0.3
	content := []gogui.View{
		gogui.Row(gogui.ContainerCfg{
			Width:   nameW,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Content: []gogui.View{gogui.Label(kit.Ellipsize(w, l.Name, st, nameW), st)},
		}),
		v.toggle(w, id, l, e),
	}
	rest := width - nameW - reach*2
	if more && (lightHue(l) || lightWarmth(l)) {
		rest -= reach
	}
	if lightDims(l) {
		content = append(content, v.slider(w, id, l, e, max(reach, rest)))
	} else {
		content = append(
			content,
			gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding}),
		)
	}
	if more && (lightHue(l) || lightWarmth(l)) {
		content = append(content, v.key(id+"-more", gogui.IconEllipsisH, func(*gogui.Window) {
			v.mu.Lock()
			v.sub.lamp = l.Entity
			v.mu.Unlock()
		}))
	}
	return gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.25),
		VAlign:  gogui.VAlignMiddle,
		Content: content,
	})
}
