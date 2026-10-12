package radar

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
)

//go:embed outlines/states.outl
var stateOutlines []byte

//go:embed outlines/counties.outl
var countyOutlines []byte

const outlineScale = 10000

type Outline struct {
	Points [][2]float32
	South  float32
	West   float32
	North  float32
	East   float32
}

var (
	states   = sync.OnceValue(func() []Outline { return mustOutlines("states", stateOutlines) })
	counties = sync.OnceValue(func() []Outline { return mustOutlines("counties", countyOutlines) })
)

func States() []Outline { return states() }

func Counties() []Outline { return counties() }

func (o Outline) Touches(v View) bool {
	south, west, north, east := v.Bounds()
	return float64(o.North) >= south && float64(o.South) <= north &&
		float64(o.East) >= west && float64(o.West) <= east
}

func mustOutlines(name string, data []byte) []Outline {
	out, err := decodeOutlines(data)
	if err != nil {
		slog.Error("reading map outlines", "set", name, "err", err)
	}
	return out
}

func decodeOutlines(data []byte) ([]Outline, error) {
	r := bytes.NewReader(data)
	magic := make([]byte, 5)
	if _, err := r.Read(magic); err != nil || string(magic) != "OUTL1" {
		return nil, errors.New("radar: not an outline file")
	}
	rings, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("radar: outline count: %w", err)
	}
	out := make([]Outline, 0, rings)
	for range rings {
		n, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, fmt.Errorf("radar: outline size: %w", err)
		}
		o := Outline{
			Points: make([][2]float32, 0, n),
			South:  math.MaxFloat32, West: math.MaxFloat32,
			North: -math.MaxFloat32, East: -math.MaxFloat32,
		}
		var x, y int64
		for range n {
			dx, err := binary.ReadVarint(r)
			if err != nil {
				return nil, fmt.Errorf("radar: outline point: %w", err)
			}
			dy, err := binary.ReadVarint(r)
			if err != nil {
				return nil, fmt.Errorf("radar: outline point: %w", err)
			}
			x, y = x+dx, y+dy
			lat, lon := float32(y)/outlineScale, float32(x)/outlineScale
			o.Points = append(o.Points, [2]float32{lat, lon})
			o.South, o.North = min(o.South, lat), max(o.North, lat)
			o.West, o.East = min(o.West, lon), max(o.East, lon)
		}
		out = append(out, o)
	}
	return out, nil
}
