package tessera

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

const (
	historyParts  = 24
	historyTicks  = 8
	historyLabels = 6
	historyStates = 7
	historyRuns   = 96
	historyWords  = 12
	historyZone   = 14 * 3600
)

type History struct {
	Entity     string
	Hours      int
	Start, End int64
	Offset     int
	Line       bool
	Ticks      []int64

	Values      []float64
	Bottom, Top float64
	Labels      []Label
	High, Low   *Extreme
	Decimals    int
	Unit        string

	Slots  int
	States []Span
	Runs   []Run
	Words  [][2]string
	Began  int64
	Active int
}

type Label struct {
	Value float64
	Text  string
}

type Extreme struct {
	Value float64
	At    int64
}

type Span struct {
	Label   string
	Color   uint32
	Seconds int64
}

type Run struct {
	Slot, State int
	Begin, End  int64
	Seconds     int64
}

type question struct {
	view   uint64
	entity string
	hours  int
}

func (s *Screen) AskHistory(entity string, hours int) {
	s.mu.Lock()
	s.asked = question{view: s.asked.view + 1, entity: entity, hours: hours}
	s.history = nil
	view := s.asked.view
	s.mu.Unlock()
	s.Tell("screen_history", map[string]string{
		"entity": entity,
		"hours":  strconv.Itoa(hours),
		"view":   strconv.FormatUint(view, 10),
	})
}

func (s *Screen) History(entity string, hours int) (History, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.history == nil || s.history.Entity != entity || s.history.Hours != hours {
		return History{}, false
	}
	return *s.history, true
}

type historyMessage struct {
	View   *uint64 `json:"view"`
	Entity string  `json:"entity"`
	Hours  int     `json:"hours"`
	Start  int64   `json:"start"`
	End    int64   `json:"end"`
	Off    int     `json:"off"`
	Kind   string  `json:"kind"`
	XT     []any   `json:"xt"`
	Values []any   `json:"values"`
	Dom    []any   `json:"dom"`
	YT     [][]any `json:"yt"`
	Hi     []any   `json:"hi"`
	Lo     []any   `json:"lo"`
	Dec    *int    `json:"dec"`
	Unit   string  `json:"unit"`
	Slots  *int    `json:"slots"`
	States [][]any `json:"states"`
	Seg    [][]any `json:"seg"`
	Words  [][]any `json:"words"`
	Began  int64   `json:"began"`
	Active *int    `json:"active"`
}

func (s *Screen) answerHistory(raw json.RawMessage) (string, bool) {
	var m historyMessage
	if json.Unmarshal(raw, &m) != nil {
		return "", false
	}
	if m.View == nil || *m.View != s.asked.view {
		return "Synced", true
	}
	if !validEntity(m.Entity) || (m.Hours != 1 && m.Hours != 24 && m.Hours != 168) {
		return "", false
	}
	if m.Entity != s.asked.entity || m.Hours != s.asked.hours {
		return "Synced", true
	}
	if m.End <= m.Start {
		return "", false
	}
	h := History{
		Entity: m.Entity,
		Hours:  m.Hours,
		Start:  m.Start,
		End:    m.End,
		Offset: min(max(m.Off, -historyZone), historyZone),
		Line:   cut(m.Kind, 12) != "timeline",
		Active: -1,
	}
	for _, x := range m.XT {
		if len(h.Ticks) == historyTicks {
			break
		}
		if at, ok := whole(x); ok {
			h.Ticks = append(h.Ticks, at)
		}
	}
	if h.Line {
		h.line(m)
	} else if !h.timeline(m) {
		return "", false
	}
	s.history = &h
	return "Synced", true
}

func (h *History) line(m historyMessage) {
	for _, v := range m.Values {
		if len(h.Values) == historyParts {
			break
		}
		h.Values = append(h.Values, real(v))
	}
	h.Bottom, h.Top = 0, 1
	if len(m.Dom) == 2 {
		h.Bottom, h.Top = orZero(real(m.Dom[0])), orZero(real(m.Dom[1]))
	}
	if !(h.Top > h.Bottom) {
		h.Top = h.Bottom + 1
	}
	for _, t := range m.YT {
		if len(h.Labels) == historyLabels {
			break
		}
		if len(t) < 2 {
			continue
		}
		if v := real(t[0]); !math.IsNaN(v) {
			text, _ := t[1].(string)
			h.Labels = append(h.Labels, Label{Value: v, Text: cut(text, 16)})
		}
	}
	h.High, h.Low = extreme(m.Hi), extreme(m.Lo)
	h.Decimals = 1
	if m.Dec != nil {
		h.Decimals = min(max(*m.Dec, 0), 4)
	}
	h.Unit = cut(m.Unit, 16)
}

func (h *History) timeline(m historyMessage) bool {
	h.Slots = historyRuns
	if m.Slots != nil {
		h.Slots = min(max(*m.Slots, 1), historyRuns)
	}
	for _, st := range m.States {
		if len(h.States) == historyStates {
			break
		}
		span := Span{}
		if len(st) > 0 {
			label, _ := st[0].(string)
			span.Label = cut(label, 24)
		}
		if len(st) > 1 {
			hex, _ := st[1].(string)
			c, _ := strconv.ParseUint(cut(hex, 8), 16, 32)
			span.Color = uint32(c)
		}
		if len(st) > 2 {
			span.Seconds, _ = whole(st[2])
		}
		h.States = append(h.States, span)
	}
	for _, item := range m.Seg {
		if len(h.Runs) == historyRuns {
			break
		}
		r := Run{State: -1}
		at := func(i int) int64 {
			if i < len(item) {
				v, _ := whole(item[i])
				return v
			}
			return 0
		}
		slot := at(0)
		if len(item) > 1 {
			if v, ok := item[1].(float64); ok {
				r.State = int(v)
			}
		}
		if slot >= int64(h.Slots) || r.State >= len(h.States) || r.State < -1 {
			return false
		}
		r.Slot = int(slot)
		r.Begin = at(2)
		r.End = max(r.Begin, at(3))
		r.Seconds = at(4)
		h.Runs = append(h.Runs, r)
	}
	for _, pair := range m.Words {
		if len(h.Words) == historyWords {
			break
		}
		var w [2]string
		for i := 0; i < 2 && i < len(pair); i++ {
			text, _ := pair[i].(string)
			w[i] = cut(text, []int{32, 24}[i])
		}
		h.Words = append(h.Words, w)
	}
	h.Began = m.Began
	if m.Active != nil && *m.Active < len(h.States) {
		h.Active = *m.Active
	}
	return true
}

func real(v any) float64 {
	f, ok := v.(float64)
	if !ok || math.IsInf(f, 0) {
		return math.NaN()
	}
	return f
}

func orZero(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return v
}

func whole(v any) (int64, bool) {
	f, ok := v.(float64)
	if !ok || f < 0 || f != math.Trunc(f) {
		return 0, false
	}
	return int64(f), true
}

func extreme(pair []any) *Extreme {
	if len(pair) == 0 {
		return nil
	}
	v := real(pair[0])
	if math.IsNaN(v) {
		return nil
	}
	e := &Extreme{Value: v}
	if len(pair) > 1 {
		e.At, _ = whole(pair[1])
	}
	return e
}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func validEntity(e string) bool {
	if e == "" || len(e) > 120 {
		return false
	}
	dot := strings.IndexByte(e, '.')
	if dot <= 0 || dot == len(e)-1 || strings.Count(e, ".") != 1 {
		return false
	}
	for _, c := range e {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '.' {
			return false
		}
	}
	return true
}
