package logd

import (
	"io"
	"log/slog"
	"time"

	"github.com/ygelfand/libcountertop/pkg/system/metrics"
)

const streamDepth = 256

type Log struct {
	Level  slog.LevelVar
	Stream *Stream

	tag      string
	bootedAt float64
	readAt   time.Time
}

func NewLog(tag string) *Log {
	return &Log{
		Stream:   NewStream(streamDepth),
		tag:      tag,
		bootedAt: metrics.Reader{}.Uptime().Value,
		readAt:   time.Now(),
	}
}

func (l *Log) Open(fallback io.Writer) *Handler {
	return NewHandler(l.tag, fallback, Options{Level: &l.Level, Uptime: l.uptime, Stream: l.Stream})
}

func (l *Log) uptime() float64 { return l.bootedAt + time.Since(l.readAt).Seconds() }
