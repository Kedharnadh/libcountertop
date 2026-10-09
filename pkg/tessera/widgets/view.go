package widgets

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

type Dependencies struct {
	UI      *kit.UI
	Screens *tessera.Set
	Perform func(ctx context.Context, action string, data, templates map[string]string) error

	HomeAfterIdle func() bool
	Logo          *ui.Logo
}

type View struct {
	o Dependencies

	mu        sync.Mutex
	page      int
	route     []string
	revision  string
	shown     string
	rested    bool
	due       time.Time
	slide     slide
	width     float32
	screen    *tessera.Screen
	last      ui.Rect
	watches   map[string]*watch
	armed     map[string]time.Time
	grips     map[string]*grip
	edits     map[string]*edit
	card      string
	radio     radio
	asked     uint64
	sub       sub
	hours     int
	code      code
	lift      float32
	rects     map[string]ui.Rect
	origin    ui.Rect
	area      ui.Rect
	camera    string
	photos    map[string]*photo
	liveAsked []liveSlot
}

func New(o Dependencies) *View {
	return &View{
		o:       o,
		slide:   slide{other: -1},
		watches: map[string]*watch{},
		armed:   map[string]time.Time{},
		grips:   map[string]*grip{},
		edits:   map[string]*edit{},
		rects:   map[string]ui.Rect{},
		photos:  map[string]*photo{},
	}
}

func (v *View) Draw(w *gogui.Window, at ui.Rect) []gogui.View {
	screen := v.pick(at)
	if screen == nil {
		return nil
	}
	at = fit(at, screen)
	layout, shown := screen.Layout()
	if !shown || len(layout.Pages) == 0 {
		return nil
	}
	g := screen.Grid()
	now := time.Now()
	f := frame{now: now, pal: v.o.UI.Palette(), fresh: screen.Fresh(now)}
	v.follow(screen, layout)
	page := v.showing(layout)
	if entity := v.showingCamera(); entity != "" {
		if t, ok := cardTile(layout, entity); ok {
			return []gogui.View{v.cameraView(w, at, screen, t, now)}
		}
		v.closeCamera(w)
	}
	var over []gogui.View
	if entity, _ := v.showingCard(); entity != "" {
		if t, ok := cardTile(layout, entity); ok {
			if v.lifted() >= 1 {
				return []gogui.View{v.cardView(w, at, layout, t, f)}
			}
			over = v.morph(at, f)
		} else {
			v.closeCard()
		}
	}
	reach := float64(style.Reach())
	margin := int(math.Round(reach * 0.5))
	gap := int(math.Round(reach * 0.25))
	band := int(math.Round(reach))
	under := gap
	foot := int(math.Round(reach * 0.6))
	strip := len(pages(layout)) > 1

	bar := ui.Rect{X: at.X + margin, Y: at.Y + margin, W: at.W - 2*margin, H: band}
	top := bar.Y + bar.H + under
	area := ui.Rect{X: at.X + margin, Y: top, W: at.W - 2*margin, H: at.Y + at.H - top - margin}
	var bottom ui.Rect
	if strip {
		area.H = at.Y + at.H - top - foot
		bottom = ui.Rect{X: at.X, Y: area.Y + area.H, W: at.W, H: foot}
	}

	views := []gogui.View{
		v.topBar(w, bar, layout, page, strip, f),
		v.pageArea(w, area, layout, g, gap, page, f),
	}
	if strip {
		views = append(views, v.bottomStrip(bottom, layout, page))
	}
	return append(views, over...)
}

func (v *View) Hint() (title, body string) {
	screen := v.pick(v.last)
	if screen == nil {
		return say.T("tessera.off"), say.T("tessera.enable")
	}
	switch screen.Stage() {
	case tessera.Unheard:
		return say.T("tessera.waiting"), say.T("tessera.install")
	case tessera.Empty:
		return say.T(
				"tessera.connected",
			), say.F(
				"tessera.empty",
				map[string]any{"Name": screen.Title()},
			)
	}
	return "", ""
}

func (v *View) pageArea(
	w *gogui.Window,
	area ui.Rect,
	l tessera.Layout,
	g schema.TesseraGrid,
	gap, page int,
	f frame,
) gogui.View {
	v.mu.Lock()
	v.width = float32(area.W)
	v.area = area
	v.mu.Unlock()
	s := v.sliding()
	inner := ui.Rect{W: area.W, H: area.H}
	tiles := v.tiles(w, l, g, inner, gap, page, int(s.offset), f)
	if s.offset != 0 && s.other >= 0 {
		shift := int(s.offset) - area.W
		if s.offset < 0 {
			shift = int(s.offset) + area.W
		}
		tiles = append(tiles, v.tiles(w, l, g, inner, gap, s.other, shift, f)...)
	}
	return kit.Placed(area, gogui.Column(gogui.ContainerCfg{
		ID:        "tessera-page",
		Width:     float32(area.W),
		Height:    float32(area.H),
		Sizing:    gogui.FixedFixed,
		Padding:   gogui.NoPadding,
		Clip:      true,
		OnGesture: v.gesture(l),
		Content:   tiles,
	}))
}

func (v *View) tiles(
	w *gogui.Window,
	l tessera.Layout,
	g schema.TesseraGrid,
	in ui.Rect,
	gap, page, shift int,
	f frame,
) []gogui.View {
	var out []gogui.View
	var slots []liveSlot
	for i, t := range l.Tiles {
		if !t.Received || t.Parent >= 0 {
			continue
		}
		c := place(t.Slot, t.Size, g)
		if c.page != page {
			continue
		}
		r := c.rect(in, g, gap)
		var pic func(float32, float32) gogui.View
		if shift == 0 {
			v.mu.Lock()
			v.rects[t.Entity] = r
			area := v.area
			v.mu.Unlock()
			if pictured(t) {
				slots = append(slots, liveSlot{
					entity: t.Entity,
					index:  i,
					at:     ui.Rect{X: area.X + r.X, Y: area.Y + r.Y, W: r.W, H: r.H},
					every:  pace(t),
					ground: groundHex(t, f.pal),
					mark:   pictureMark(t),
				})
				pic = v.livePicture(i)
			}
		}
		r.X += shift
		out = append(
			out,
			kit.Placed(r, v.tile(w, fmt.Sprintf("tessera-%d-%d", page, i), t, l, r, c, f, pic)),
		)
	}
	if shift == 0 {
		v.wantLive(w, slots, f)
	}
	return out
}

func (v *View) Rested() {
	v.mu.Lock()
	v.rested = true
	v.mu.Unlock()
}

func (v *View) Turns() bool { return v.o.Screens.Turns() }

func (v *View) pick(at ui.Rect) *tessera.Screen {
	screen := v.o.Screens.For(at.W, at.H)
	v.mu.Lock()
	defer v.mu.Unlock()
	v.last = at
	if screen != v.screen {
		v.screen, v.revision, v.shown, v.route, v.card = screen, "", "", nil, ""
		if screen != nil {
			v.asked = screen.Asked().Ask
		}
	}
	return screen
}

func fit(at ui.Rect, s *tessera.Screen) ui.Rect {
	w, h := s.Canvas()
	if w <= 0 || h <= 0 || (w >= h) == (at.W >= at.H) {
		return at
	}
	if at.W*h > at.H*w {
		fw := at.H * w / h
		return ui.Rect{X: at.X + (at.W-fw)/2, Y: at.Y, W: fw, H: at.H}
	}
	fh := at.W * h / w
	return ui.Rect{X: at.X, Y: at.Y + (at.H-fh)/2, W: at.W, H: fh}
}

type frame struct {
	now   time.Time
	pal   theme.Theme
	fresh bool
}
