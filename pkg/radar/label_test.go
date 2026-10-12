package radar

import (
	"image"
	"math"
	"testing"
)

func TestLabelsDecode(t *testing.T) {
	tests := []struct {
		set   string
		names []Label
		want  string
		lat   float64
		lon   float64
	}{
		{"states", StateNames(), "Massachusetts", 42.3, -71.8},
		{"counties", CountyNames(), "Suffolk", 42.3, -71.0},
		{"cities", CityNames(), "Boston", 42.36, -71.06},
	}
	for _, tt := range tests {
		if len(tt.names) == 0 {
			t.Fatalf("%s: no labels", tt.set)
		}
		found := false
		for _, l := range tt.names {
			if l.Name == tt.want && math.Abs(float64(l.Lat)-tt.lat) < 0.5 &&
				math.Abs(float64(l.Lon)-tt.lon) < 0.5 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: no %q near %v,%v", tt.set, tt.want, tt.lat, tt.lon)
		}
	}
}

func TestCitiesRankedBiggestFirst(t *testing.T) {
	cities := CityNames()
	for i := 1; i < len(cities); i++ {
		if cities[i].Rank < cities[i-1].Rank {
			t.Fatalf(
				"city %d %q rank %d after rank %d",
				i,
				cities[i].Name,
				cities[i].Rank,
				cities[i-1].Rank,
			)
		}
	}
}

func TestLabelIn(t *testing.T) {
	v := View{Lat: 42.36, Lon: -71.06, Zoom: 8.8, Size: image.Pt(1280, 800)}
	if !(Label{Lat: 42.36, Lon: -71.06}).In(v) {
		t.Error("center label not in view")
	}
	if (Label{Lat: 34.05, Lon: -118.24}).In(v) {
		t.Error("Los Angeles in a Boston view")
	}
}

func TestDecodeLabelsRejectsTruncated(t *testing.T) {
	if _, err := decodeLabels(cityLabels[:len(cityLabels)/2]); err == nil {
		t.Error("truncated label file decoded")
	}
	if _, err := decodeLabels([]byte("OUTL1")); err == nil {
		t.Error("outline file decoded as labels")
	}
}
