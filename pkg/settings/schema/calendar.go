package schema

import (
	"slices"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Calendar struct {
	Entities string       `json:"entities"`
	Span     CalendarSpan `json:"span"`
}

type CalendarSpan string

const (
	Span3Days  CalendarSpan = "3"
	Span7Days  CalendarSpan = "7"
	Span14Days CalendarSpan = "14"
)

func CalendarSpans() []CalendarSpan { return []CalendarSpan{Span3Days, Span7Days, Span14Days} }

func (s CalendarSpan) Label() string { return say.F("calendar.days", map[string]any{"N": s.Days()}) }

func (s CalendarSpan) Days() int {
	switch s {
	case Span3Days:
		return 3
	case Span14Days:
		return 14
	}
	return 7
}

func DefaultCalendar() Calendar { return Calendar{Span: Span7Days} }

func (c Calendar) List() []string {
	var out []string
	for e := range strings.SplitSeq(c.Entities, ",") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func (c Calendar) Has(entity string) bool { return slices.Contains(c.List(), entity) }

type CalendarWriter struct{ s storage.Section[Calendar] }

func NewCalendarWriter(s storage.Section[Calendar]) CalendarWriter {
	return CalendarWriter{s: s}
}

func (w CalendarWriter) Entities(list []string) error {
	return w.s.Update(func(c *Calendar) { c.Entities = strings.Join(list, ",") })
}

func (w CalendarWriter) Span(v CalendarSpan) error {
	return w.s.Update(func(c *Calendar) { c.Span = v })
}
