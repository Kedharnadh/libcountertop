package tessera

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

const samplesMost = 24

type Layout struct {
	Revision  string
	Title     string
	Home      int
	Clock24   bool
	Inbox     string
	Keepalive int
	Numbers   Numbers
	Pages     []Page
	Tiles     []Tile
	Saver     json.RawMessage
}

type Page struct {
	ID          string
	Title       string
	HomeControl bool
	Excluded    bool
	Bar         []BarItem
}

type BarItem struct {
	Kind  string `json:"k"`
	Icon  string `json:"i"`
	Text  string `json:"t"`
	Epoch int64  `json:"e"`
	Color string `json:"c"`
	Weak  int    `json:"a"`
}

var barKinds = map[string]bool{
	"clock": true, "analog": true, "date": true, "text": true,
	"ago": true, "wifi": true, "link": true, "battery": true,
}

func bar(raw json.RawMessage) ([]BarItem, bool) {
	var items []BarItem
	if json.Unmarshal(raw, &items) != nil || len(items) > barItems {
		return nil, false
	}
	for _, it := range items {
		if !it.valid() {
			return nil, false
		}
	}
	return items, true
}

func (it BarItem) valid() bool { return barKinds[it.Kind] && (it.Kind != "ago" || it.Epoch != 0) }

type Tile struct {
	Entity     string
	Size       string
	Display    string
	Tap        string
	Sub        string
	Controls   string
	Inline     string
	Guard      string
	Refresh    int
	Slot       int
	Parent     int
	Key        int
	Name       string
	State      string
	Icon       string
	Background string
	Options    json.RawMessage
	Attributes json.RawMessage
	Extra      json.RawMessage
	History    json.RawMessage
	Received   bool
}

func (l Layout) clone() Layout {
	l.Pages = append([]Page(nil), l.Pages...)
	for i := range l.Pages {
		l.Pages[i].Bar = append([]BarItem(nil), l.Pages[i].Bar...)
	}
	l.Tiles = append([]Tile(nil), l.Tiles...)
	return l
}

func (t Tile) Domain() string {
	domain, _, _ := strings.Cut(t.Entity, ".")
	return domain
}

func (t Tile) attributes() map[string]any {
	var a map[string]any
	_ = json.Unmarshal(t.Attributes, &a)
	return a
}

func (t Tile) Text(key string) string {
	switch v := t.attributes()[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func (t Tile) Number(key string) (float64, bool) {
	v, ok := t.attributes()[key].(float64)
	return v, ok
}

func (t Tile) Bool(key string) (bool, bool) {
	v, ok := t.attributes()[key].(bool)
	return v, ok
}

func (t Tile) Numbers(key string) []float64 {
	list, _ := t.attributes()[key].([]any)
	out := make([]float64, 0, len(list))
	for _, v := range list {
		f, ok := v.(float64)
		if !ok {
			return nil
		}
		out = append(out, f)
	}
	return out
}

func (t Tile) Texts(key string) []string {
	list, _ := t.attributes()[key].([]any)
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (t Tile) extra() map[string]any {
	var x map[string]any
	_ = json.Unmarshal(t.Extra, &x)
	return x
}

func (t Tile) ExtraText(key string) string {
	v, _ := t.extra()[key].(string)
	return v
}

func (t Tile) ExtraNumber(key string) (float64, bool) {
	v, ok := t.extra()[key].(float64)
	return v, ok
}

func (t Tile) ExtraList(key string) []map[string]any {
	list, _ := t.extra()[key].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, v := range list {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func (t Tile) ExtraNumbers(key string) []float64 {
	list, _ := t.extra()[key].([]any)
	out := make([]float64, len(list))
	for i, v := range list {
		out[i], _ = v.(float64)
	}
	return out
}

func (t Tile) ExtraTexts(key string) []string {
	list, _ := t.extra()[key].([]any)
	out := make([]string, 0, len(list))
	for _, v := range list {
		s, _ := v.(string)
		out = append(out, s)
	}
	return out
}

func (t Tile) Samples() []float64 {
	var h struct {
		Values []*float64 `json:"values"`
	}
	if json.Unmarshal(t.History, &h) != nil {
		return nil
	}
	out := make([]float64, 0, min(len(h.Values), samplesMost))
	for _, v := range h.Values[:min(len(h.Values), samplesMost)] {
		if v == nil {
			out = append(out, math.NaN())
			continue
		}
		out = append(out, *v)
	}
	return out
}

func (t Tile) ExtraBool(key string) bool {
	v, _ := t.extra()[key].(bool)
	return v
}

type Action struct {
	Service   string
	Data      map[string]string
	Templates map[string]string
}

func (t Tile) Action() Action {
	var o struct {
		Act struct {
			S string      `json:"s"`
			D [][2]string `json:"d"`
			T [][2]string `json:"t"`
		} `json:"act"`
	}
	_ = json.Unmarshal(t.Options, &o)
	a := Action{Service: o.Act.S, Data: map[string]string{}, Templates: map[string]string{}}
	for _, p := range o.Act.D {
		a.Data[p[0]] = p[1]
	}
	for _, p := range o.Act.T {
		a.Templates[p[0]] = p[1]
	}
	return a
}
