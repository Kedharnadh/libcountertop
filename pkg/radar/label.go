package radar

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	_ "embed"
)

//go:embed outlines/states.labl
var stateLabels []byte

//go:embed outlines/counties.labl
var countyLabels []byte

//go:embed outlines/cities.labl
var cityLabels []byte

type Label struct {
	Name string
	Lat  float32
	Lon  float32
	Rank int
}

var (
	stateNames  = sync.OnceValue(func() []Label { return mustLabels("states", stateLabels) })
	countyNames = sync.OnceValue(func() []Label { return mustLabels("counties", countyLabels) })
	cityNames   = sync.OnceValue(func() []Label { return mustLabels("cities", cityLabels) })
)

func StateNames() []Label { return stateNames() }

func CountyNames() []Label { return countyNames() }

func CityNames() []Label { return cityNames() }

func (l Label) In(v View) bool {
	south, west, north, east := v.Bounds()
	lat, lon := float64(l.Lat), float64(l.Lon)
	return lat >= south && lat <= north && lon >= west && lon <= east
}

func mustLabels(name string, data []byte) []Label {
	out, err := decodeLabels(data)
	if err != nil {
		slog.Error("reading map labels", "set", name, "err", err)
	}
	return out
}

func decodeLabels(data []byte) ([]Label, error) {
	r := bytes.NewReader(data)
	magic := make([]byte, 5)
	if _, err := io.ReadFull(r, magic); err != nil || string(magic) != "LABL1" {
		return nil, errors.New("radar: not a label file")
	}
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("radar: label count: %w", err)
	}
	out := make([]Label, 0, n)
	for range n {
		size, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, fmt.Errorf("radar: label name: %w", err)
		}
		if size > uint64(r.Len()) {
			return nil, errors.New("radar: label name past end")
		}
		name := make([]byte, size)
		if _, err := io.ReadFull(r, name); err != nil {
			return nil, fmt.Errorf("radar: label name: %w", err)
		}
		lon, err := binary.ReadVarint(r)
		if err != nil {
			return nil, fmt.Errorf("radar: label point: %w", err)
		}
		lat, err := binary.ReadVarint(r)
		if err != nil {
			return nil, fmt.Errorf("radar: label point: %w", err)
		}
		rank, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("radar: label rank: %w", err)
		}
		out = append(out, Label{
			Name: string(name),
			Lat:  float32(lat) / outlineScale,
			Lon:  float32(lon) / outlineScale,
			Rank: int(rank),
		})
	}
	return out, nil
}
