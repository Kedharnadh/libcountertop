package cameraview

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"
	"golang.org/x/sync/errgroup"

	"github.com/ygelfand/libcountertop/pkg/display/apps"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/homecontrol"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	listEvery   = 4 * time.Minute
	fetchWait   = 10 * time.Second
	liveRetry   = time.Second
	swipeFar    = 80
	aspect      = 9.0 / 16.0
	fetchAtMost = 3
)

var scaleShare = map[int]float64{1: 0.2, 2: 0.25, 3: 1.0 / 3, 4: 0.5, 5: 1}

type Source interface {
	Cameras(context.Context) ([]homeassistant.Camera, error)
}

type Site interface{ URL() string }

type Options struct {
	Shell    *shell.Shell
	Source   Source
	Site     Site
	Settings storage.Section[schema.Cameras]
	Client   *http.Client
	Picker   func() shell.View
}

type Grid struct{}

func (*Grid) Covers() bool { return true }

type Live struct{}

func (*Live) Covers() bool { return true }

type frame struct {
	key string
	src string
	w   int
	h   int
}

type Cameras struct {
	o     Options
	grid  *Grid
	live  *Live
	knobs *setting.Controls[schema.Cameras]

	mu       sync.Mutex
	list     []homeassistant.Camera
	listErr  error
	frames   map[string]frame
	gen      uint64
	page     int
	pages    int
	shown    []string
	tile     image.Point
	screen   image.Point
	panned   float32
	watching string
	liveShot frame
	stopGrid context.CancelFunc
	stopLive context.CancelFunc
}

func New(o Options) *Cameras {
	c := &Cameras{o: o, grid: &Grid{}, live: &Live{}, frames: map[string]frame{}}
	c.knobs = &setting.Controls[schema.Cameras]{
		Table: NewTable(),
		Read:  o.Settings.Read,
		Save:  func(s Setting, v string) error { return setting.Store(o.Settings, s, v) },
	}
	o.Shell.Changed.Listen(func(shell.Change) { c.follow() })
	return c
}

func (c *Cameras) Controls() *setting.Controls[schema.Cameras] { return c.knobs }

func (c *Cameras) App() apps.App {
	return apps.App{
		ID:   schema.AppCameras,
		Name: func() string { return say.T("cameras.app") },
		Icon: "mdi:cctv",
		Open: func() { c.o.Shell.Push(c.grid) },
	}
}

func (c *Cameras) Screen(v shell.View) *navigation.Screen {
	switch v {
	case c.grid:
		return &navigation.Screen{View: v, Clear: true, Build: c.buildGrid}
	case c.live:
		return &navigation.Screen{View: v, Clear: true, Fixed: true, Build: c.buildLive}
	}
	return nil
}

func (c *Cameras) follow() {
	grid := c.o.Shell.Visible(c.grid)
	live := c.o.Shell.Visible(c.live)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case grid && c.stopGrid == nil:
		ctx, stop := context.WithCancel(context.Background())
		c.stopGrid = stop
		safe.Go("cameras grid", func() { c.runGrid(ctx) })
	case !grid && c.stopGrid != nil:
		c.stopGrid()
		c.stopGrid = nil
	}
	if !live && c.stopLive != nil {
		c.stopLive()
		c.stopLive = nil
	}
}

func (c *Cameras) runGrid(ctx context.Context) {
	cfg := c.o.Settings.Read()
	c.refreshList(ctx)
	c.fetchShown(ctx)
	list := time.NewTicker(listEvery)
	defer list.Stop()
	refresh, stopRefresh := every(cfg.Refresh)
	defer stopRefresh()
	rotate, stopRotate := every(cfg.Rotate)
	defer stopRotate()
	for {
		select {
		case <-ctx.Done():
			return
		case <-list.C:
			c.refreshList(ctx)
		case <-refresh:
			c.fetchShown(ctx)
		case <-rotate:
			c.turn(1)
			c.fetchShown(ctx)
		}
	}
}

func every(seconds int) (<-chan time.Time, func()) {
	if seconds <= 0 {
		return nil, func() {}
	}
	t := time.NewTicker(time.Duration(seconds) * time.Second)
	return t.C, t.Stop
}

func (c *Cameras) refreshList(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, fetchWait)
	defer cancel()
	list, err := c.o.Source.Cameras(ctx)
	c.mu.Lock()
	if err == nil {
		c.list = list
	}
	c.listErr = err
	c.mu.Unlock()
	if err != nil {
		slog.Warn("listing cameras failed", "err", err)
	}
	c.o.Shell.Redraw()
}

func (c *Cameras) picked() []homeassistant.Camera {
	pick := c.o.Settings.Read().Pick
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []homeassistant.Camera
	for _, cam := range c.list {
		if homecontrol.Includes(pick, cam.Entity) {
			out = append(out, cam)
		}
	}
	return out
}

func (c *Cameras) camera(id string) (homeassistant.Camera, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	i := slices.IndexFunc(c.list, func(cam homeassistant.Camera) bool { return cam.ID == id })
	if i < 0 {
		return homeassistant.Camera{}, false
	}
	return c.list[i], true
}

func (c *Cameras) fetchShown(ctx context.Context) {
	c.mu.Lock()
	ids, size := slices.Clone(c.shown), c.tile
	c.mu.Unlock()
	base := c.o.Site.URL()
	var g errgroup.Group
	g.SetLimit(fetchAtMost)
	for _, id := range ids {
		cam, ok := c.camera(id)
		if !ok || cam.Picture == "" {
			continue
		}
		g.Go(func() error {
			ctx, cancel := context.WithTimeout(ctx, fetchWait)
			defer cancel()
			img, err := snapshot(ctx, c.o.Client, base, cam.Picture, size.X, size.Y)
			if err != nil {
				slog.Debug("camera snapshot failed", "camera", id, "err", err)
				return nil
			}
			c.keep(id, img)
			return nil
		})
	}
	_ = g.Wait()
	c.o.Shell.Redraw()
}

func (c *Cameras) keep(id string, img *image.NRGBA) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	key := fmt.Sprintf("camera/%s/%d", id, c.gen)
	src := gogui.UseImage(key, img.Rect.Dx(), img.Rect.Dy(), img.Pix)
	if old, ok := c.frames[id]; ok {
		gogui.DropImage(old.key)
	}
	c.frames[id] = frame{key: key, src: src, w: img.Rect.Dx(), h: img.Rect.Dy()}
}

func (c *Cameras) turn(by int) {
	c.mu.Lock()
	if c.pages > 1 {
		c.page = ((c.page+by)%c.pages + c.pages) % c.pages
	}
	c.mu.Unlock()
	c.o.Shell.Redraw()
}

func Layout(n, scale int, w, h float64) (cols, rows int) {
	if n <= 0 || w <= 0 || h <= 0 {
		return 1, 1
	}
	share, ok := scaleShare[scale]
	if !ok {
		share = scaleShare[3]
	}
	cols = max(1, int(math.Floor(1/share+1e-9)))
	rows = max(1, int(math.Floor(h/(w/float64(cols)*aspect)+1e-9)))
	if n < cols*rows {
		cols = min(cols, n)
		rows = int(math.Ceil(float64(n) / float64(cols)))
	}
	return cols, rows
}

func (c *Cameras) buildGrid(w *gogui.Window) gogui.View {
	vw, vh := w.WindowSize()
	cams := c.picked()
	if len(cams) == 0 {
		return c.note(c.emptyNote())
	}
	cols, rows := Layout(len(cams), c.o.Settings.Read().Scale, float64(vw), float64(vh))
	per := cols * rows
	tw, th := float32(vw)/float32(cols), float32(vh)/float32(rows)

	c.mu.Lock()
	c.pages = (len(cams) + per - 1) / per
	c.page = min(c.page, c.pages-1)
	from := c.page * per
	shown := cams[from:min(from+per, len(cams))]
	ids := make([]string, len(shown))
	for i, cam := range shown {
		ids[i] = cam.ID
	}
	moved := !slices.Equal(ids, c.shown)
	c.shown, c.tile, c.screen = ids, image.Point{X: int(tw), Y: int(th)}, image.Point{X: vw, Y: vh}
	frames := make([]frame, len(shown))
	for i, cam := range shown {
		frames[i] = c.frames[cam.ID]
	}
	page, pages := c.page, c.pages
	c.mu.Unlock()
	if moved {
		safe.Go("cameras page", func() { c.fetchShown(context.Background()) })
	}

	var lines []gogui.View
	for r := range rows {
		var row []gogui.View
		for k := range cols {
			i := r*cols + k
			if i >= len(shown) {
				break
			}
			row = append(row, c.tileView(shown[i], frames[i], tw, th))
		}
		lines = append(lines, gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Content: row,
		}))
	}
	if pages > 1 {
		lines = append(lines, gogui.Label(
			say.F("cameras.page", map[string]any{"N": page + 1, "Of": pages}),
			widgets.Secondary(),
		))
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:        "cameras",
		Sizing:    gogui.FillFill,
		Padding:   gogui.NoPadding,
		Color:     gogui.RGB(0, 0, 0),
		Content:   lines,
		OnGesture: c.swipe,
	})
}

func (c *Cameras) swipe(e gogui.EventCtx) {
	if e.Event.GestureType != gogui.GesturePan && e.Event.GestureType != gogui.GestureSwipe {
		return
	}
	c.mu.Lock()
	switch e.Event.GesturePhase {
	case gogui.GesturePhaseBegan:
		c.panned = e.Event.GestureDX
	case gogui.GesturePhaseChanged:
		c.panned += e.Event.GestureDX
	case gogui.GesturePhaseEnded:
		far := c.panned
		c.panned = 0
		c.mu.Unlock()
		switch {
		case far <= -swipeFar:
			c.turn(1)
		case far >= swipeFar:
			c.turn(-1)
		}
		e.Consume()
		return
	}
	c.mu.Unlock()
	e.Consume()
}

func (c *Cameras) tileView(cam homeassistant.Camera, f frame, tw, th float32) gogui.View {
	var content []gogui.View
	if f.src != "" {
		content = append(content, gogui.Image(gogui.ImageCfg{
			Src:    f.src,
			Width:  float32(f.w),
			Height: float32(f.h),
		}))
	} else {
		content = append(content, gogui.Label(cam.Name, widgets.Secondary()))
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:      "camera-" + cam.ID,
		Width:   tw,
		Height:  th,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: content,
		OnClick: func(e gogui.EventCtx) {
			c.watch(cam.ID)
			e.Consume()
		},
	})
}

func (c *Cameras) watch(id string) {
	c.mu.Lock()
	if c.stopLive != nil {
		c.stopLive()
	}
	ctx, stop := context.WithCancel(context.Background())
	c.stopLive, c.watching = stop, id
	c.mu.Unlock()
	c.o.Shell.Push(c.live)
	safe.Go("camera live", func() { c.runLive(ctx, id) })
}

func (c *Cameras) runLive(ctx context.Context, id string) {
	for ctx.Err() == nil {
		cam, ok := c.camera(id)
		if !ok {
			return
		}
		c.mu.Lock()
		size := c.screen
		c.mu.Unlock()
		err := stream(
			ctx,
			c.o.Client,
			c.o.Site.URL(),
			cam.Picture,
			size.X,
			size.Y,
			func(img *image.NRGBA) {
				c.showLive(img)
			},
		)
		if ctx.Err() != nil {
			return
		}
		slog.Debug("camera live stream ended", "camera", id, "err", err)
		snap, sctx := context.WithTimeout(ctx, fetchWait)
		if img, err := snapshot(
			snap,
			c.o.Client,
			c.o.Site.URL(),
			cam.Picture,
			size.X,
			size.Y,
		); err == nil {
			c.showLive(img)
		}
		sctx()
		select {
		case <-ctx.Done():
			return
		case <-time.After(liveRetry):
		}
	}
}

func (c *Cameras) showLive(img *image.NRGBA) {
	c.mu.Lock()
	c.gen++
	key := fmt.Sprintf("camera/live/%d", c.gen)
	src := gogui.UseImage(key, img.Rect.Dx(), img.Rect.Dy(), img.Pix)
	if c.liveShot.key != "" {
		gogui.DropImage(c.liveShot.key)
	}
	c.liveShot = frame{key: key, src: src, w: img.Rect.Dx(), h: img.Rect.Dy()}
	c.mu.Unlock()
	c.o.Shell.Redraw()
}

func (c *Cameras) buildLive(*gogui.Window) gogui.View {
	c.mu.Lock()
	f, id := c.liveShot, c.watching
	c.mu.Unlock()
	var content []gogui.View
	if f.src != "" {
		content = append(content, gogui.Image(gogui.ImageCfg{
			Src:    f.src,
			Width:  float32(f.w),
			Height: float32(f.h),
		}))
	} else if cam, ok := c.camera(id); ok {
		content = append(content, gogui.Label(cam.Name, widgets.Secondary()))
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:      "camera-live",
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Color:   gogui.RGB(0, 0, 0),
		Content: content,
		OnClick: func(e gogui.EventCtx) {
			c.o.Shell.Remove(c.live)
			c.mu.Lock()
			if c.liveShot.key != "" {
				gogui.DropImage(c.liveShot.key)
			}
			c.liveShot = frame{}
			c.mu.Unlock()
			e.Consume()
		},
	})
}

func (c *Cameras) emptyNote() string {
	c.mu.Lock()
	err, listed := c.listErr, c.list != nil
	c.mu.Unlock()
	switch {
	case c.o.Site.URL() == "":
		return say.T("cameras.nosite")
	case err != nil:
		return say.T("cameras.failed")
	case !listed:
		return say.T("cameras.loading")
	}
	return say.T("cameras.none")
}

func (c *Cameras) note(text string) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{gogui.Label(text, widgets.Secondary())},
	})
}

func (c *Cameras) Page() *shell.Page {
	return &shell.Page{Title: say.T("cameras.app"), Build: func() ([]menu.Row, []func(int)) {
		cfg := c.o.Settings.Read()
		scale := menu.Steps(schema.CameraScaleMost - schema.CameraScaleLeast + 1)
		refresh, rotate := menu.Steps(len(refreshSteps)), menu.Steps(len(rotateSteps))
		rows := []menu.Row{
			{
				Label: say.T("cameras.pick"),
				Kind:  menu.Chevron,
				Value: homecontrol.Summary(cfg.Pick),
			},
			{
				Label: say.T("cameras.setting.scale"),
				Kind:  menu.Slider,
				Level: scale.Level(cfg.Scale - schema.CameraScaleLeast),
				Snap:  scale.Snap,
				Value: strconv.Itoa(cfg.Scale),
			},
			{
				Label: say.T("cameras.setting.refresh"),
				Kind:  menu.Slider,
				Level: refresh.Level(
					slices.Index(refreshSteps, nearest(refreshSteps, cfg.Refresh)),
				),
				Snap:  refresh.Snap,
				Value: seconds(cfg.Refresh),
			},
			{
				Label: say.T("cameras.setting.rotate"),
				Kind:  menu.Slider,
				Level: rotate.Level(slices.Index(rotateSteps, nearest(rotateSteps, cfg.Rotate))),
				Snap:  rotate.Snap,
				Value: seconds(cfg.Rotate),
			},
			{
				Label: say.T("cameras.setting.url"),
				Kind:  menu.Field,
				Value: cfg.URL,
				Hint:  c.o.Site.URL(),
				Save:  func(v string) { c.change("url", v) },
			},
		}
		taps := []func(int){
			func(int) { c.o.Shell.Push(c.o.Picker()) },
			func(level int) {
				c.change("scale", strconv.Itoa(scale.Index(level)+schema.CameraScaleLeast))
			},
			func(level int) { c.change("refresh", strconv.Itoa(refreshSteps[refresh.Index(level)])) },
			func(level int) { c.change("rotate", strconv.Itoa(rotateSteps[rotate.Index(level)])) },
			nil,
		}
		return rows, taps
	}}
}

func (c *Cameras) change(name, value string) {
	if err := c.knobs.Change(name, value); err != nil {
		slog.Error("saving a setting failed", "setting", "cameras."+name, "err", err)
	}
}

func nearest(steps []int, v int) int {
	best := steps[0]
	for _, s := range steps {
		if abs(s-v) < abs(best-v) {
			best = s
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func seconds(n int) string {
	if n == 0 {
		return say.T("cameras.off")
	}
	return say.F("cameras.seconds", map[string]any{"N": n})
}
