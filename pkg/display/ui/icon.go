package ui

import (
	"image"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

type Icon []byte

var (
	iconMu    sync.Mutex
	iconCache = map[iconKey]*image.Alpha{}
	iconBytes int
)

type iconKey struct {
	icon string
	w, h int
}

const iconBudget = 4 << 20

func evict(keep iconKey) {
	for at, m := range iconCache {
		if at.w == keep.w && at.h == keep.h {
			continue
		}
		iconBytes -= len(m.Pix)
		delete(iconCache, at)
	}
	for at, img := range colorCache {
		if at.w == keep.w && at.h == keep.h {
			continue
		}
		if img != nil {
			iconBytes -= len(img.Pix)
		}
		delete(colorCache, at)
	}

	if iconBytes > iconBudget {
		iconCache = map[iconKey]*image.Alpha{}
		colorCache = map[iconKey]*image.RGBA{}
		iconBytes = 0
	}
}

func mask(icon Icon, w, h int) *image.Alpha {
	at := iconKey{icon: string(icon), w: w, h: h}

	iconMu.Lock()
	defer iconMu.Unlock()

	if m, ok := iconCache[at]; ok {
		return m
	}

	rgba, err := raster(icon, w, h)
	if err != nil {
		blank := image.NewAlpha(image.Rect(0, 0, w, h))

		if iconBytes+len(blank.Pix) > iconBudget {
			evict(at)
		}
		iconCache[at] = blank
		iconBytes += len(blank.Pix)
		return blank
	}

	m := image.NewAlpha(rgba.Bounds())
	for i, p := 0, 0; i < len(m.Pix); i, p = i+1, p+4 {
		m.Pix[i] = rgba.Pix[p+3]
	}

	if iconBytes+len(m.Pix) > iconBudget {
		evict(at)
	}

	iconCache[at] = m
	iconBytes += len(m.Pix)
	return m
}

func DrawIcon(s Surface, icon Icon, r Rect, fg, on theme.Color) {
	size := min(r.W, r.H)
	if size <= 0 {
		return
	}

	stamp(s, mask(icon, size, size),
		r.X+(r.W-size)/2, r.Y+(r.H-size)/2, size, size, fg, on)
}

func DrawIconFilling(s Surface, icon Icon, r Rect, fg, on theme.Color) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	stamp(s, mask(icon, r.W, r.H), r.X, r.Y, r.W, r.H, fg, on)
}

func stamp(s Surface, m *image.Alpha, x0, y0, mw, mh int, fg, on theme.Color) {
	w, h := s.Size()

	for y := range mh {
		for x := range mw {
			a := m.Pix[y*m.Stride+x]
			if a == 0 {
				continue
			}

			px, py := x0+x, y0+y
			if px < 0 || py < 0 || px >= w || py >= h {
				continue
			}
			s.Set(px, py, on.Mix(fg, a))
		}
	}
}
