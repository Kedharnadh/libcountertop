// The conversion is adapted from golang.org/x/exp/shiny/materialdesign/icons/gen.go, which is
// Copyright 2016 The Go Authors and BSD licensed.
package mark

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/exp/shiny/iconvg"
	"golang.org/x/image/math/f32"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

const size = 48

type svg struct {
	ViewBox string   `xml:"viewBox,attr"`
	Paths   []path   `xml:"path"`
	Circles []circle `xml:"circle"`
}

type path struct {
	D string `xml:"d,attr"`
}

// IconVG has no circles, so they become a pair of half arcs.
type circle struct {
	Cx float32 `xml:"cx,attr"`
	Cy float32 `xml:"cy,attr"`
	R  float32 `xml:"r,attr"`
}

func Convert(data []byte) (ui.Icon, error) { return convert(data, false) }

func Shaped(data []byte) (ui.Icon, error) { return convert(data, true) }

func convert(data []byte, shaped bool) (ui.Icon, error) {
	var in svg
	if err := xml.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("mark: reading the svg: %w", err)
	}

	side, extent, offset, err := box(in.ViewBox)
	if err != nil {
		return nil, err
	}
	if len(in.Paths) == 0 && len(in.Circles) == 0 {
		return nil, fmt.Errorf("mark: the svg has no filled paths")
	}

	if !shaped {
		extent = f32.Vec2{size, size}
	}

	var enc iconvg.Encoder
	enc.Reset(iconvg.Metadata{
		ViewBox: iconvg.Rectangle{
			Min: f32.Vec2{-extent[0] / 2, -extent[1] / 2},
			Max: f32.Vec2{+extent[0] / 2, +extent[1] / 2},
		},
		Palette: iconvg.DefaultPalette,
	})

	started := false
	for _, p := range in.Paths {
		if p.D == "" {
			continue
		}
		if err := draw(&enc, p.D, side, offset, &started); err != nil {
			return nil, err
		}
	}

	for _, c := range in.Circles {
		cx := c.Cx*size/side - size/2 - offset[0]
		cy := c.Cy*size/side - size/2 - offset[1]
		r := c.R * size / side

		if !started {
			started = true
			enc.StartPath(0, cx-r, cy)
		} else {
			enc.ClosePathAbsMoveTo(cx-r, cy)
		}

		// An SVG arc whose endpoints coincide is degenerate.
		enc.RelArcTo(r, r, 0, false, true, +2*r, 0)
		enc.RelArcTo(r, r, 0, false, true, -2*r, 0)
	}

	if !started {
		return nil, fmt.Errorf("mark: the svg drew nothing")
	}
	enc.ClosePathEndPath()

	out, err := enc.Bytes()
	if err != nil {
		return nil, fmt.Errorf("mark: encoding: %w", err)
	}
	return ui.Icon(out), nil
}

func box(viewBox string) (side float32, extent, offset f32.Vec2, err error) {
	if viewBox == "" {
		return 24, f32.Vec2{size, size}, f32.Vec2{}, nil
	}

	fields := strings.Fields(strings.ReplaceAll(viewBox, ",", " "))
	if len(fields) != 4 {
		return 0, extent, offset, fmt.Errorf("mark: viewBox %q is not four numbers", viewBox)
	}

	var at [4]float32
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return 0, extent, offset, fmt.Errorf("mark: viewBox %q: %w", viewBox, err)
		}
		at[i] = float32(v)
	}

	w, h := at[2], at[3]
	if w <= 0 || h <= 0 {
		return 0, extent, offset, fmt.Errorf("mark: viewBox %q has no area", viewBox)
	}

	side = max(w, h)
	extent = f32.Vec2{w * size / side, h * size / side}
	offset = f32.Vec2{
		(at[0] + (w-side)/2) * size / side,
		(at[1] + (h-side)/2) * size / side,
	}
	return side, extent, offset, nil
}

func draw(enc *iconvg.Encoder, data string, side float32, offset f32.Vec2, started *bool) error {
	data = strings.TrimSuffix(strings.TrimSpace(data), "z")
	r := strings.NewReader(strings.ReplaceAll(data, ",", " "))

	var args [6]float32
	op, relative := byte(0), false

	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch {
		case b == ' ' || b == '\n' || b == '\t':
			continue
		case 'A' <= b && b <= 'Z':
			op, relative = b, false
		case 'a' <= b && b <= 'z':
			op, relative = b, true
		default:
			_ = r.UnreadByte()
		}

		n := 0
		switch op {
		case 'H', 'h', 'V', 'v':
			n = 1
		case 'L', 'l', 'T', 't', 'M', 'm':
			n = 2
		case 'Q', 'q', 'S', 's':
			n = 4
		case 'C', 'c':
			n = 6
		case 'Z', 'z':
		default:
			return fmt.Errorf("mark: unknown path command %q", string(b))
		}

		scan(&args, r, n)
		normalize(&args, n, op, side, offset, relative)

		switch op {
		case 'L':
			enc.AbsLineTo(args[0], args[1])
		case 'l':
			enc.RelLineTo(args[0], args[1])
		case 'T':
			enc.AbsSmoothQuadTo(args[0], args[1])
		case 't':
			enc.RelSmoothQuadTo(args[0], args[1])
		case 'Q':
			enc.AbsQuadTo(args[0], args[1], args[2], args[3])
		case 'q':
			enc.RelQuadTo(args[0], args[1], args[2], args[3])
		case 'S':
			enc.AbsSmoothCubeTo(args[0], args[1], args[2], args[3])
		case 's':
			enc.RelSmoothCubeTo(args[0], args[1], args[2], args[3])
		case 'C':
			enc.AbsCubeTo(args[0], args[1], args[2], args[3], args[4], args[5])
		case 'c':
			enc.RelCubeTo(args[0], args[1], args[2], args[3], args[4], args[5])
		case 'H':
			enc.AbsHLineTo(args[0])
		case 'h':
			enc.RelHLineTo(args[0])
		case 'V':
			enc.AbsVLineTo(args[0])
		case 'v':
			enc.RelVLineTo(args[0])
		case 'M':
			if !*started {
				*started = true
				enc.StartPath(0, args[0], args[1])
			} else {
				enc.ClosePathAbsMoveTo(args[0], args[1])
			}
		case 'm':
			if !*started {
				*started = true
				enc.StartPath(0, args[0], args[1])
			} else {
				enc.ClosePathRelMoveTo(args[0], args[1])
			}
		}
	}
}

func scan(args *[6]float32, r *strings.Reader, n int) {
	for i := range n {
		for {
			if b, _ := r.ReadByte(); b != ' ' {
				_ = r.UnreadByte()
				break
			}
		}
		_, _ = fmt.Fscanf(r, "%f", &args[i])
	}
}

func normalize(args *[6]float32, n int, op byte, side float32, offset f32.Vec2, relative bool) {
	for i := range n {
		args[i] *= size / side
		if relative {
			continue
		}
		args[i] -= size / 2

		switch {
		case n != 1:
			args[i] -= offset[i&0x01]
		case op == 'H':
			args[i] -= offset[0]
		case op == 'V':
			args[i] -= offset[1]
		}
	}
}

func Must(data []byte) ui.Icon {
	icon, err := Convert(data)
	if err != nil {
		panic(err)
	}
	return icon
}
