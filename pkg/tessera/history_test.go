package tessera

import (
	"math"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
)

func committed(t *testing.T) *sender {
	t.Helper()
	c := &sender{t: t, s: screen()}
	c.session = c.send(map[string]any{"op": "hello", "request": "34f13e8fb3066a17"})["session"].(string)
	for _, m := range []map[string]any{
		{"op": "begin", "tiles": 0, "pages": 1, "home": 0, "title": "", "keepalive": 60,
			"inbox": "kitchen", "numbers": "comma", "percent_space": true},
		{"op": "page", "p": 0, "id": "00000000000000b1", "title": "", "home_control": false, "excluded": false, "items": []any{}},
		{"op": "commit"},
	} {
		c.send(m)
	}
	return c
}

func TestOnlyTheLatestHistoryQuestionIsAnswered(t *testing.T) {
	c := committed(t)
	var told []component.Event
	c.s.Told.Listen(func(e component.Event) { told = append(told, e) })

	c.s.AskHistory("sensor.fridge", 24)
	c.s.AskHistory("sensor.fridge", 1)
	if len(told) != 2 || told[1].Name != "esphome.screen_history" ||
		told[1].Data["view"] != "2" || told[1].Data["hours"] != "1" || told[1].Data["inbox"] != "kitchen" {
		t.Fatalf("asked %+v", told)
	}

	line := func(view, hours int) map[string]any {
		return map[string]any{
			"op":     "history",
			"view":   view,
			"entity": "sensor.fridge",
			"hours":  hours,
			"start":  100,
			"end":    3700,
			"off":    -18000,
			"kind":   "line",
			"values": []any{36.5, nil, 37.25},
			"dom":    []any{35, 39},
			"yt": []any{
				[]any{35, "35°"},
				[]any{39, "39°"},
			},
			"hi":   []any{37.25, 3600},
			"lo":   []any{36.5, 200},
			"dec":  2,
			"unit": "°F",
			"xt":   []any{100, 1900, 3700},
		}
	}
	c.send(line(1, 24))
	if _, ok := c.s.History("sensor.fridge", 24); ok {
		t.Fatal("an answer to a question no longer asked replaced the history")
	}
	c.send(line(2, 24))
	if _, ok := c.s.History("sensor.fridge", 24); ok {
		t.Fatal("an answer for another range replaced the history")
	}
	c.send(line(2, 1))
	h, ok := c.s.History("sensor.fridge", 1)
	if !ok {
		t.Fatal("the answer to the latest question was dropped")
	}
	if !h.Line || len(h.Values) != 3 || !math.IsNaN(h.Values[1]) || h.Values[2] != 37.25 {
		t.Errorf("values %+v", h.Values)
	}
	if h.Bottom != 35 || h.Top != 39 || len(h.Labels) != 2 || h.Labels[1].Text != "39°" {
		t.Errorf("scale %v..%v labels %+v", h.Bottom, h.Top, h.Labels)
	}
	if h.High == nil || h.High.At != 3600 || h.Decimals != 2 || h.Unit != "°F" ||
		len(h.Ticks) != 3 {
		t.Errorf("history %+v", h)
	}
}

func TestATimelineWithABadRunIsRefusedWhole(t *testing.T) {
	c := committed(t)
	c.s.AskHistory("binary_sensor.door", 24)
	timeline := func(state int) map[string]any {
		return map[string]any{
			"op": "history", "view": 1, "entity": "binary_sensor.door", "hours": 24,
			"start": 0, "end": 86400, "kind": "timeline", "slots": 48,
			"states": []any{[]any{"Open", "f44336", 600}, []any{"Closed", "9e9e9e", 85800}},
			"seg":    []any{[]any{0, 1, 0, 1800, 1800}, []any{3, state, 5400, 6000, 600}},
			"words":  []any{[]any{"times", "3"}}, "began": 6000, "active": 1,
		}
	}
	c.send(timeline(5))
	if _, ok := c.s.History("binary_sensor.door", 24); ok {
		t.Fatal("a run naming a state that does not exist was kept")
	}
	c.send(timeline(0))
	h, ok := c.s.History("binary_sensor.door", 24)
	if !ok || h.Line || h.Slots != 48 || len(h.States) != 2 || h.States[0].Color != 0xf44336 {
		t.Fatalf("timeline %+v", h)
	}
	if len(h.Runs) != 2 || h.Runs[1].State != 0 || h.Active != 1 ||
		h.Words[0] != [2]string{"times", "3"} {
		t.Errorf("runs %+v active %d words %+v", h.Runs, h.Active, h.Words)
	}
}
