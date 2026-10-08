package screencmd

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/display/panel"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
)

func Shot(d *panel.Driver, path string) error {
	img, err := Composed(d)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func Composed(d *panel.Driver) (*image.RGBA, error) {
	if c := d.Helper(); c != nil {
		pix, w, h, err := c.ScreenRead()
		if err != nil {
			return nil, err
		}
		o := d.Orientation()
		vw, vh := o.Size(w, h)
		img := image.NewRGBA(image.Rect(0, 0, vw, vh))
		for vy := range vh {
			for vx := range vw {
				x, y := o.Project(w, h, vx, vy)
				copy(img.Pix[(vy*vw+vx)*4:(vy*vw+vx)*4+4], pix[(y*w+x)*4:(y*w+x)*4+4])
			}
		}
		return img, nil
	}

	pixels, w, h, err := d.Shot()
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range w * h {
		img.Set(i%w, i/w, color.RGBA{R: pixels[i*3], G: pixels[i*3+1], B: pixels[i*3+2], A: 0xff})
	}
	return img, nil
}

func Stack(s *shell.Shell) string {
	views := s.Stack()
	if len(views) == 0 {
		return "empty"
	}
	var b strings.Builder
	for i := len(views) - 1; i >= 0; i-- {
		v := views[i]
		where := ""
		if i == len(views)-1 {
			where = " (top)"
		}
		title := ""
		if p, ok := v.View.(*shell.Page); ok {
			title = fmt.Sprintf(" %q", p.Title)
		}
		fmt.Fprintf(
			&b,
			"%d %T%s held=%v covers=%v%s\n",
			i,
			v.View,
			title,
			v.Held,
			v.View.Covers(),
			where,
		)
	}
	return b.String()
}
