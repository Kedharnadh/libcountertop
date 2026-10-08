package logd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type Options struct {
	Level  slog.Leveler
	Uptime func() float64
	Stream *Stream
}

type Handler struct {
	conn     *conn
	fallback io.Writer
	opts     Options
	attrs    []slog.Attr
	group    string

	mu *sync.Mutex
}

type Line struct {
	Level slog.Level
	Text  string

	Dropped uint64
}

type Stream struct {
	lines   chan Line
	dropped atomic.Uint64
}

func NewStream(depth int) *Stream { return &Stream{lines: make(chan Line, depth)} }

func (s *Stream) Lines() <-chan Line { return s.lines }

func (s *Stream) publish(l Line) {
	l.Dropped = s.dropped.Swap(0)

	select {
	case s.lines <- l:
		return
	default:
	}

	select {
	case <-s.lines:
		s.dropped.Add(1)
	default:
	}

	select {
	case s.lines <- l:
	default:
		s.dropped.Add(1 + l.Dropped)
	}
}

func NewHandler(tag string, fallback io.Writer, opts Options) *Handler {
	h := &Handler{fallback: fallback, opts: opts, mu: &sync.Mutex{}}

	c, err := dial(tag)
	if err != nil {
		fmt.Fprintf(fallback, "logcat unavailable: %v\n", err)
		return h
	}
	h.conn = c
	return h
}

func (h *Handler) Close() error {
	if h.conn == nil {
		return nil
	}
	return h.conn.Close()
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	floor := slog.LevelInfo
	if h.opts.Level != nil {
		floor = h.opts.Level.Level()
	}
	return l >= floor
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := *h
	out.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &out
}

func (h *Handler) WithGroup(name string) slog.Handler {
	out := *h
	out.group = name
	return &out
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	if h.opts.Uptime != nil {
		fmt.Fprintf(&b, "[%8.2f] ", h.opts.Uptime())
	}
	b.WriteString(r.Message)

	for _, a := range h.attrs {
		appendAttr(&b, h.group, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(&b, h.group, a)
		return true
	})

	line := b.String()

	if h.opts.Stream != nil {
		h.opts.Stream.publish(Line{Level: r.Level, Text: line})
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.fallback != nil {
		fmt.Fprintf(h.fallback, "%s %s\n", Letter(r.Level), line)
	}
	if h.conn == nil {
		return nil
	}
	return h.conn.write(Priority(r.Level), line)
}

func appendAttr(b *strings.Builder, group string, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	b.WriteByte(' ')
	if group != "" {
		b.WriteString(group)
		b.WriteByte('.')
	}
	b.WriteString(a.Key)
	b.WriteByte('=')
	b.WriteString(value(a.Value))
}

func value(v slog.Value) string {
	s := v.String()
	if strings.ContainsAny(s, " \t=\"") {
		return strconv.Quote(s)
	}
	return s
}
