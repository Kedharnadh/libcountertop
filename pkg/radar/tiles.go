package radar

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

const (
	tileMost      = 2 << 20
	progressFirst = 100 * time.Millisecond
	progressEvery = 200 * time.Millisecond
)

type lifetime struct {
	keep  bool
	until time.Time
}

type tiles struct {
	client *http.Client
	agent  string
	limit  int
	cache  *cache

	fetched atomic.Int64
	reused  atomic.Int64
}

func (t *tiles) get(ctx context.Context, url string, life lifetime) (image.Image, error) {
	if body, ok := t.cache.get(url, life.keep); ok {
		if img, _, err := image.Decode(bytes.NewReader(body)); err == nil {
			t.reused.Add(1)
			return img, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", t.agent)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("radar: %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, tileMost))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	t.fetched.Add(1)
	t.cache.put(url, body, life.keep, life.until)
	return img, nil
}

func (t *tiles) layer(
	ctx context.Context,
	v View,
	most int,
	life lifetime,
	url func(z, x, y int) string,
	progress func(*image.RGBA),
) (*image.RGBA, error) {
	z := int(math.Min(math.Ceil(v.Zoom), float64(most)))
	scale := math.Exp2(v.Zoom - float64(z))
	cx, cy := worldPixel(v.Lat, v.Lon, float64(z))
	w := float64(v.Size.X) / scale
	h := float64(v.Size.Y) / scale
	x0, y0 := cx-w/2, cy-h/2
	tx0, ty0 := int(math.Floor(x0/tileSize)), int(math.Floor(y0/tileSize))
	tx1, ty1 := int(math.Floor((x0+w)/tileSize)), int(math.Floor((y0+h)/tileSize))
	span := (tx1 - tx0 + 1) * tileSize
	rows := (ty1 - ty0 + 1) * tileSize
	mosaic := image.NewRGBA(image.Rect(0, 0, span, rows))
	count := 1 << z
	src := image.Rect(0, 0, int(math.Round(w)), int(math.Round(h))).
		Add(image.Pt(int(math.Round(x0))-tx0*tileSize, int(math.Round(y0))-ty0*tileSize))
	scaled := func(from *image.RGBA) *image.RGBA {
		out := image.NewRGBA(image.Rect(0, 0, v.Size.X, v.Size.Y))
		xdraw.ApproxBiLinear.Scale(out, out.Bounds(), from, src, draw.Src, nil)
		return out
	}
	var (
		mu       sync.Mutex
		placed   int
		failed   error
		shown    time.Time
		drawing  bool
		started  = time.Now()
		snapshot = func() *image.RGBA {
			return &image.RGBA{
				Pix:    slices.Clone(mosaic.Pix),
				Stride: mosaic.Stride,
				Rect:   mosaic.Rect,
			}
		}
	)
	var g errgroup.Group
	g.SetLimit(max(t.limit, 1))
	for ty := ty0; ty <= ty1; ty++ {
		if ty < 0 || ty >= count {
			continue
		}
		for tx := tx0; tx <= tx1; tx++ {
			g.Go(func() error {
				img, err := t.get(ctx, url(z, ((tx%count)+count)%count, ty), life)
				mu.Lock()
				if err != nil {
					if failed == nil {
						failed = err
					}
					mu.Unlock()
					return nil
				}
				place(mosaic, image.Pt((tx-tx0)*tileSize, (ty-ty0)*tileSize), img)
				placed++
				var copied *image.RGBA
				if progress != nil && !drawing && time.Since(started) > progressFirst &&
					time.Since(shown) > progressEvery {
					drawing, copied = true, snapshot()
				}
				mu.Unlock()
				if copied != nil {
					progress(scaled(copied))
					mu.Lock()
					drawing, shown = false, time.Now()
					mu.Unlock()
				}
				return nil
			})
		}
	}
	_ = g.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if placed == 0 && failed != nil {
		return nil, failed
	}
	if failed != nil {
		slog.Debug("radar tiles missing", "err", failed)
	}
	return scaled(mosaic), nil
}

func place(dst *image.RGBA, at image.Point, img image.Image) {
	p, ok := img.(*image.Paletted)
	if !ok {
		b := img.Bounds()
		draw.Draw(dst, image.Rectangle{Min: at, Max: at.Add(b.Size())}, img, b.Min, draw.Src)
		return
	}
	var lut [256][4]uint8
	for i, c := range p.Palette {
		r, g, b, a := c.RGBA()
		lut[i] = [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	}
	b := p.Bounds()
	for y := 0; y < b.Dy() && at.Y+y < dst.Rect.Dy(); y++ {
		src := p.Pix[y*p.Stride:]
		row := dst.Pix[(at.Y+y)*dst.Stride+at.X*4:]
		for x := 0; x < b.Dx() && at.X+x < dst.Rect.Dx(); x++ {
			copy(row[x*4:x*4+4], lut[src[x]][:])
		}
	}
}
