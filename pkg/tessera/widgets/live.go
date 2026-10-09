package widgets

import (
	"encoding/json"
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const liveSize = 160

type liveSlot struct {
	entity string
	index  int
	at     ui.Rect
	every  int
	ground string
	mark   string
}

func pictured(t tessera.Tile) bool {
	d := t.Domain()
	switch t.Display {
	case "live":
		return d == "camera" || d == "image"
	case "cover":
		return d == "media_player" && t.ExtraText("pic") != ""
	case "map":
		return d == "person" || t.Entity == "screen.map"
	case "favorite":
		return favorite(t) && t.ExtraText("fm") != ""
	}
	return false
}

func pictureMark(t tessera.Tile) string {
	switch t.Display {
	case "cover":
		return t.ExtraText("pic")
	case "map":
		return t.ExtraText("mk")
	case "favorite":
		return t.ExtraText("fm")
	}
	return ""
}

func pace(t tessera.Tile) int {
	if t.Display == "live" {
		return t.Refresh
	}
	return 0
}

func groundHex(t tessera.Tile, pal theme.Theme) string {
	if g := ground(t.Background, pal); g != nil && g.A > 0 {
		return fmt.Sprintf("%02x%02x%02x", g.R, g.G, g.B)
	}
	return fmt.Sprintf("%02x%02x%02x", pal.Surface.R, pal.Surface.G, pal.Surface.B)
}

func (v *View) wantLive(w *gogui.Window, frames []liveSlot, f frame) {
	if v.want(tessera.PictureLive, fmt.Sprint(frames)) {
		v.mu.Lock()
		v.liveAsked = frames
		v.mu.Unlock()
	}
	v.mu.Lock()
	s := v.screen
	v.mu.Unlock()
	if len(frames) == 0 || s == nil {
		return
	}
	every := 0
	entities, idx, grounds, atlas := []string{}, []string{}, []string{}, [][6]int{}
	for _, fr := range frames {
		if fr.every > 0 && (every == 0 || fr.every < every) {
			every = fr.every
		}
		entities = append(entities, fr.entity)
		idx = append(idx, strconv.Itoa(fr.index))
		grounds = append(grounds, fr.ground)
		atlas = append(atlas, [6]int{fr.at.X, fr.at.Y, fr.at.W, fr.at.H, 0, 0})
	}
	frameJSON, _ := json.Marshal(atlas)
	dark := "0"
	if f.pal.Dark {
		dark = "1"
	}
	fresh := time.Hour
	if every > 0 {
		fresh = time.Duration(every) * time.Second
	}
	v.keepFresh(w, s, tessera.PictureLive, "", fresh, func() {
		s.AskPicture(tessera.PictureLive, map[string]string{
			"tiles": strings.Join(entities, ","),
			"idx":   strings.Join(idx, ","),
			"size":  strconv.Itoa(liveSize),
			"bg":    strings.Join(grounds, ","),
			"dark":  dark,
			"atlas": string(frameJSON),
		})
	}, f.now)
}

func (v *View) livePicture(index int) func(width, height float32) gogui.View {
	v.mu.Lock()
	p := v.photos[tessera.PictureLive]
	var img image.Image
	if p != nil {
		img = p.img
	}
	var at ui.Rect
	found := false
	for _, fr := range v.liveAsked {
		if fr.index == index {
			at, found = fr.at, true
		}
	}
	v.mu.Unlock()
	if img == nil || !found {
		return nil
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil
	}
	crop := sub.SubImage(image.Rect(at.X, at.Y, at.X+at.W, at.Y+at.H))
	if crop.Bounds().Empty() {
		return nil
	}
	src := v.source(tessera.PictureLive, strconv.Itoa(index), crop)
	return func(width, height float32) gogui.View {
		pw, ph := fitted(crop, ui.Rect{W: int(width), H: int(height)})
		return kit.Picture(src, pw, ph)
	}
}

const (
	coverLeast = 48
	coverMost  = 320
)

func (v *View) cover(w *gogui.Window, t tessera.Tile, width float32, now time.Time) gogui.View {
	mark := t.ExtraText("pic")
	if mark == "" {
		return nil
	}
	size := min(max(int(width*0.6), coverLeast), coverMost)
	bg := fmt.Sprintf(
		"%02x%02x%02x",
		v.o.UI.Palette().Surface.R,
		v.o.UI.Palette().Surface.G,
		v.o.UI.Palette().Surface.B,
	)
	v.want(
		tessera.PictureCover,
		strings.Join([]string{t.Entity, mark, strconv.Itoa(size), bg}, "|"),
	)
	v.mu.Lock()
	s := v.screen
	v.mu.Unlock()
	if s == nil {
		return nil
	}
	img := v.keepFresh(w, s, tessera.PictureCover, t.Entity, time.Hour, func() {
		s.AskPicture(
			tessera.PictureCover,
			map[string]string{"entity": t.Entity, "size": strconv.Itoa(size), "bg": bg},
		)
	}, now)
	if img == nil {
		return nil
	}
	pw, ph := fitted(img, ui.Rect{W: size, H: size})
	return kit.Picture(v.source(tessera.PictureCover, t.Entity, img), pw, ph)
}
