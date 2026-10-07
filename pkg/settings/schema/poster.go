package schema

import (
	"strings"
	"time"

	"github.com/ygelfand/libcountertop/pkg/say"
)

type Poster struct {
	Enabled bool        `json:"enabled"`
	Every   PosterEvery `json:"every"`
	Server  string      `json:"server"`
	Key     string      `json:"key"`
	Albums  string      `json:"albums"`
	Tags    string      `json:"tags"`
	Last    string      `json:"last"`
}

func DefaultPoster() Poster { return Poster{Every: PosterHourly} }

type PosterEvery string

const (
	PosterNever  PosterEvery = "never"
	PosterWake   PosterEvery = "wake"
	PosterHourly PosterEvery = "hourly"
	PosterDaily  PosterEvery = "daily"
)

func PosterEveries() []PosterEvery {
	return []PosterEvery{PosterNever, PosterWake, PosterHourly, PosterDaily}
}

func (p PosterEvery) Label() string {
	switch p {
	case PosterNever:
		return say.T("poster.never")
	case PosterWake:
		return say.T("poster.wake")
	case PosterDaily:
		return say.T("poster.daily")
	}
	return say.T("poster.hourly")
}

func (p PosterEvery) Period() time.Duration {
	switch p {
	case PosterHourly:
		return time.Hour
	case PosterDaily:
		return 24 * time.Hour
	}
	return 0
}

func Names(list string) []string {
	var out []string
	for s := range strings.SplitSeq(list, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
