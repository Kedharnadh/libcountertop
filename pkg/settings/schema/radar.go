package schema

import (
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/say"
)

type RadarSource string

const (
	RadarAuto       RadarSource = "auto"
	RadarNWS        RadarSource = "nws"
	RadarRainViewer RadarSource = "rainviewer"
)

func RadarSources() []RadarSource { return []RadarSource{RadarAuto, RadarNWS, RadarRainViewer} }

func (s RadarSource) Label() string { return say.T("radar.source." + string(s)) }

type RadarRange string

const (
	Range50  RadarRange = "50"
	Range100 RadarRange = "100"
	Range200 RadarRange = "200"
	Range300 RadarRange = "300"
	Range400 RadarRange = "400"
	Range600 RadarRange = "600"
	Range800 RadarRange = "800"
)

func RadarRanges() []RadarRange {
	return []RadarRange{Range50, Range100, Range200, Range300, Range400, Range600, Range800}
}

func (r RadarRange) Miles() int {
	if n, err := strconv.Atoi(string(r)); err == nil && n > 0 {
		return n
	}
	return 200
}

func (r RadarRange) Label() string { return say.F("radar.miles", map[string]any{"N": r.Miles()}) }
