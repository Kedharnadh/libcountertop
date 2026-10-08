package visual

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"path"
	"sync"

	xdraw "golang.org/x/image/draw"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Thumbs struct {
	fsys fs.FS
	dir  string

	mu     sync.Mutex
	thumbs map[string]*ui.Image
	fitted map[string]*image.RGBA
}

func NewThumbs(fsys fs.FS, dir string) *Thumbs {
	return &Thumbs{
		fsys:   fsys,
		dir:    dir,
		thumbs: map[string]*ui.Image{},
		fitted: map[string]*image.RGBA{},
	}
}

func thumbName(k Kind, portrait bool) string {
	if portrait {
		return string(k) + "-portrait"
	}
	return string(k)
}

func (t *Thumbs) read(name string) ([]byte, error) {
	return fs.ReadFile(t.fsys, path.Join(t.dir, name+".jpg"))
}

func (t *Thumbs) Thumbnail(k Kind, portrait bool) *ui.Image {
	name := thumbName(k, portrait)
	t.mu.Lock()
	defer t.mu.Unlock()
	if img, ok := t.thumbs[name]; ok {
		return img
	}
	b, err := t.read(name)
	var img *ui.Image
	if err == nil {
		img, _ = ui.Decode(b)
	}
	t.thumbs[name] = img
	return img
}

func (t *Thumbs) Fit(k Kind, w, h int) *image.RGBA {
	if w <= 0 || h <= 0 {
		return nil
	}
	name := thumbName(k, h > w)
	key := fmt.Sprintf("%s@%dx%d", name, w, h)
	t.mu.Lock()
	defer t.mu.Unlock()
	if img, ok := t.fitted[key]; ok {
		return img
	}
	var out *image.RGBA
	if b, err := t.read(name); err == nil {
		if src, err := jpeg.Decode(bytes.NewReader(b)); err == nil {
			sb := src.Bounds()
			sw, sh := sb.Dx(), sb.Dy()
			if sw*h > sh*w {
				sw = sh * w / h
			} else {
				sh = sw * h / w
			}
			crop := image.Rect(0, 0, sw, sh).
				Add(sb.Min).
				Add(image.Pt((sb.Dx()-sw)/2, (sb.Dy()-sh)/2))
			out = image.NewRGBA(image.Rect(0, 0, w, h))
			xdraw.ApproxBiLinear.Scale(out, out.Bounds(), src, crop, xdraw.Src, nil)
		}
	}
	t.fitted[key] = out
	return out
}
