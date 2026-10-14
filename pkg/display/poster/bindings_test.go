package poster

import (
	"time"

	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

var posterConfig config.Poster

type posterWriter struct{}

func (posterWriter) Enabled(
	v bool,
) error {
	posterConfig.Enabled = v
	return nil
}

func (posterWriter) Every(v config.PosterEvery) error         { posterConfig.Every = v; return nil }
func (posterWriter) Read() config.Poster                      { return posterConfig }
func (posterWriter) Update(change func(*config.Poster)) error { change(&posterConfig); return nil }

type posterTouch struct{}

func (posterTouch) Since() time.Duration { return 0 }
func testOptions() Dependencies {
	return Dependencies{
		Settings: posterWriter{},
		Touch:    posterTouch{},
	}
}
