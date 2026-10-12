package tessera

import (
	"encoding/json"
	"hash/fnv"
	"log/slog"
	"strings"
	"time"
)

const mostPayload = 4096

type message struct {
	V         json.RawMessage `json:"v"`
	Op        string          `json:"op"`
	Request   string          `json:"request"`
	Session   string          `json:"session"`
	Rev       string          `json:"rev"`
	Seq       *uint32         `json:"seq"`
	Empty     bool            `json:"empty"`
	Tiles     json.RawMessage `json:"tiles"`
	Pages     json.RawMessage `json:"pages"`
	Home      *int            `json:"home"`
	Title     *string         `json:"title"`
	Keepalive *int            `json:"keepalive"`
	Clock24   *bool           `json:"clock_24h"`
	Numbers   *string         `json:"numbers"`
	GroupMin  *int            `json:"group_min"`
	Percent   *bool           `json:"percent_space"`
	Inbox     string          `json:"inbox"`
	P         *int            `json:"p"`
	ID        string          `json:"id"`
	Items     json.RawMessage `json:"items"`
	Control   *bool           `json:"home_control"`
	Excluded  *bool           `json:"excluded"`
	I         *int            `json:"i"`
	Entity    string          `json:"entity"`
	Slot      *int            `json:"slot"`
	In        *int            `json:"in"`
	K         json.RawMessage `json:"k"`
	O         json.RawMessage `json:"o"`
	A         json.RawMessage `json:"a"`
	X         json.RawMessage `json:"x"`
	State     *string         `json:"state"`
	Name      *string         `json:"name"`
	Icon      string          `json:"icon"`
	History   json.RawMessage `json:"history"`
	Base      string          `json:"base"`
	Targets   []int           `json:"targets"`
	Item      json.RawMessage `json:"item"`
	W         json.RawMessage `json:"w"`

	raw json.RawMessage
}

func (s *Screen) Receive(payload string) map[string]any {
	s.Regrid()
	s.mu.Lock()
	status := s.receive(payload)
	if strings.HasPrefix(status, "Error") || status == "Resend needed" {
		slog.Warn("tessera refused a message", "status", status, "payload", clip(payload))
	}
	answer := s.answer(status)
	var kept *Layout
	if s.dirty && s.session.active {
		l := s.layout.clone()
		kept, s.dirty = &l, false
	}
	s.mu.Unlock()
	if kept != nil {
		s.keep(*kept)
	}
	s.Changed.Emit(struct{}{})
	return answer
}

func (s *Screen) receive(payload string) string {
	s.memory = false
	if len(payload) > mostPayload {
		return "Error: message too large"
	}
	var m message
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return "Error: invalid message"
	}
	m.raw = json.RawMessage(payload)
	switch string(m.V) {
	case "2":
	case "1":
		return "Configuration problem. Update add-on."
	default:
		return "Error: protocol version"
	}
	s.memory = m.Op == "hello" || m.Op == "ping"
	if m.Op == "hello" {
		request, ok := key(m.Request)
		if !ok {
			return "Error: invalid message"
		}
		s.heard = true
		return "Session:" + keyText(s.session.grant(request, random()))
	}
	lease, okLease := key(m.Session)
	revision, okRev := key(m.Rev)
	if !okLease || !okRev || m.Seq == nil {
		return "Error: invalid message"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(payload))
	hash := h.Sum32()
	switch s.session.packet(lease, *m.Seq, hash) {
	case reject:
		return "Error: obsolete message"
	case duplicate:
		return s.loading()
	}
	status, ok := s.apply(m, revision)
	if ok {
		s.session.accepted(*m.Seq, hash)
		s.fed = time.Now()
	}
	if !ok && status == "" {
		status = "Error: invalid message"
	}
	return status
}

func (s *Screen) loading() string {
	if s.session.active {
		return "Synced"
	}
	return "Loading tiles"
}

func count(raw json.RawMessage) (int, bool) {
	var n int
	if err := json.Unmarshal(raw, &n); err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func (s *Screen) apply(m message, revision uint64) (string, bool) {
	switch m.Op {
	case "begin":
		return s.begin(m, revision)
	case "appearance":
		return s.appearance(m, revision)
	}
	if !s.session.matches(revision) {
		return "Resend needed", false
	}
	switch m.Op {
	case "ping":
		if s.session.active {
			return "Synced", true
		}
		return "Resend needed", true
	case "bar_value":
		return s.barValue(m)
	case "page", "bar":
		return s.page(m)
	case "commit":
		if s.session.active {
			return "Synced", true
		}
		if !s.session.commit() {
			return "Error: incomplete layout", false
		}
		s.dirty, s.restored = true, false
		slog.Info(
			"tessera layout applied",
			"pages",
			len(s.layout.Pages),
			"tiles",
			len(s.layout.Tiles),
		)
		return "Synced", true
	case "tile":
		return s.tile(m, true)
	}
	if !s.session.active {
		return "Resend needed", false
	}
	switch m.Op {
	case "state":
		return s.tile(m, false)
	case "saver":
		s.layout.Saver = m.raw
		return "Synced", true
	case "history":
		return s.answerHistory(m.raw)
	case "camera":
		return s.answerPicture(m.raw)
	case "options":
		return s.answerNames(m.raw)
	case "browse":
		return "Synced", true
	}
	return "", false
}

func (s *Screen) begin(m message, revision uint64) (string, bool) {
	tiles, okTiles := count(m.Tiles)
	pages, okPages := count(m.Pages)
	if !okTiles || !okPages || m.Home == nil || m.Title == nil || len(*m.Title) > 96 ||
		pages == 0 || pages > maxPages || tiles > maxTiles || *m.Home < 0 || *m.Home >= pages ||
		m.Keepalive == nil || *m.Keepalive < 5 || *m.Keepalive > 3600 {
		return "", false
	}
	candidate := s.session
	switch candidate.begin(revision, pages, tiles) {
	case refused:
		return "", false
	case unchanged:
		s.session = candidate
		return "Synced", true
	}
	s.session = candidate
	s.restored = false
	s.layout = Layout{
		Revision:  keyText(revision),
		Title:     *m.Title,
		Home:      *m.Home,
		Inbox:     m.Inbox,
		Keepalive: *m.Keepalive,
		Clock24:   s.layout.Clock24,
		Numbers:   s.layout.Numbers.from(m),
		Pages:     make([]Page, pages),
		Tiles:     make([]Tile, tiles),
	}
	if m.Clock24 != nil {
		s.layout.Clock24 = *m.Clock24
	}
	return "Loading tiles", true
}

func (s *Screen) appearance(m message, revision uint64) (string, bool) {
	base, ok := key(m.Base)
	if !ok || !s.session.active || !s.session.matches(base) || m.Title == nil {
		return "", false
	}
	var pages []struct {
		P     int    `json:"p"`
		Title string `json:"title"`
	}
	var tiles []struct {
		I          int    `json:"i"`
		Name       string `json:"name"`
		Background string `json:"background"`
	}
	if json.Unmarshal(m.Pages, &pages) != nil || json.Unmarshal(m.Tiles, &tiles) != nil {
		return "", false
	}
	for _, p := range pages {
		if p.P < 0 || p.P >= len(s.layout.Pages) {
			return "", false
		}
	}
	for _, t := range tiles {
		if t.I < 0 || t.I >= len(s.layout.Tiles) {
			return "", false
		}
	}
	s.layout.Title = *m.Title
	for _, p := range pages {
		s.layout.Pages[p.P].Title = p.Title
	}
	for _, t := range tiles {
		s.layout.Tiles[t.I].Name, s.layout.Tiles[t.I].Background = t.Name, t.Background
	}
	s.session.revision = revision
	s.layout.Revision = keyText(revision)
	s.dirty = true
	return "Synced", true
}

func (s *Screen) barValue(m message) (string, bool) {
	if !s.session.active || len(m.Targets) == 0 {
		return "", false
	}
	stride := wireItems
	if len(m.W) > 0 && json.Unmarshal(m.W, &stride) != nil {
		return "", false
	}
	if stride < 1 {
		return "", false
	}
	var item BarItem
	if json.Unmarshal(m.Item, &item) != nil || !item.valid() {
		return "", false
	}
	for _, t := range m.Targets {
		if t < 0 || t/stride >= len(s.layout.Pages) ||
			t%stride >= len(s.layout.Pages[t/stride].Bar) {
			return "", false
		}
	}
	for _, t := range m.Targets {
		s.layout.Pages[t/stride].Bar[t%stride] = item
	}
	return "Synced", true
}

func (s *Screen) page(m message) (string, bool) {
	if m.P == nil || *m.P < 0 || *m.P >= len(s.layout.Pages) {
		return "", false
	}
	index := *m.P
	items, ok := bar(m.Items)
	if !ok {
		return "", false
	}
	if m.Op == "bar" {
		if !s.session.active {
			return "", false
		}
		s.layout.Pages[index].Bar = items
		s.dirty = true
		return "Synced", true
	}
	if s.session.active || s.session.receivedPage(index) || m.Control == nil || m.Excluded == nil ||
		m.Title == nil || len(*m.Title) > 96 {
		return "", false
	}
	if _, ok := key(m.ID); !ok {
		return "", false
	}
	for i, p := range s.layout.Pages {
		if s.session.receivedPage(i) && p.ID == m.ID {
			return "", false
		}
	}
	if !s.session.page(index) {
		return "", false
	}
	s.layout.Pages[index] = Page{
		ID:          m.ID,
		Title:       *m.Title,
		HomeControl: *m.Control,
		Excluded:    *m.Excluded,
		Bar:         items,
	}
	return s.loading(), true
}

func (s *Screen) tile(m message, initial bool) (string, bool) {
	if m.I == nil || m.State == nil || m.Name == nil || len(m.A) == 0 || m.A[0] != '{' {
		return "", false
	}
	index := *m.I
	if index < 0 || index >= len(s.layout.Tiles) {
		return "", false
	}
	t := &s.layout.Tiles[index]
	if initial {
		place, keyed := count(m.K)
		placed := m.Slot != nil || (m.In != nil && keyed)
		if s.session.active || t.Received || m.Entity == "" || !placed || len(m.O) == 0 ||
			!s.session.tile(index) {
			return "", false
		}
		var o struct {
			Size       string `json:"size"`
			Display    string `json:"display"`
			Tap        string `json:"tap"`
			Background string `json:"background"`
			Icon       string `json:"icon"`
			Sub        string `json:"sub"`
			Controls   string `json:"controls"`
			Inline     string `json:"inline"`
			Guard      string `json:"guard"`
			Flow       string `json:"flow"`
			Refresh    int    `json:"refresh"`
		}
		_ = json.Unmarshal(m.O, &o)
		refresh := refreshDefault
		if o.Refresh >= refreshLeast && o.Refresh <= refreshMost {
			refresh = o.Refresh
		}
		*t = Tile{
			Entity:     m.Entity,
			Size:       o.Size,
			Display:    o.Display,
			Tap:        o.Tap,
			Background: o.Background,
			Icon:       o.Icon,
			Sub:        o.Sub,
			Controls:   o.Controls,
			Inline:     o.Inline,
			Guard:      o.Guard,
			Flow:       o.Flow,
			Refresh:    refresh,
			Options:    m.O,
			Parent:     -1,
			Key:        -1,
			Received:   true,
		}
		if m.Slot != nil {
			t.Slot = *m.Slot
		} else {
			t.Parent, t.Key = *m.In, place
		}
	} else {
		if len(m.O) != 0 || m.Slot != nil || !t.Received || t.Entity != m.Entity {
			return "Error: outdated tile or configuration in state", false
		}
		if m.Icon != "" {
			t.Icon = m.Icon
		}
	}
	t.Name, t.State = *m.Name, *m.State
	t.Attributes, t.Extra, t.History = m.A, m.X, m.History
	return s.loading(), true
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
