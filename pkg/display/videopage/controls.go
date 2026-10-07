package videopage

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	media "github.com/ygelfand/libcountertop/pkg/media/source"
)

type Controls interface {
	media.Source
	Seek(to time.Duration)
	CanSeek() bool
}

type Mark struct {
	From, To time.Duration
	Color    theme.Color
}

type Marked interface {
	Marks() []Mark
}

const linger = 4 * time.Second
