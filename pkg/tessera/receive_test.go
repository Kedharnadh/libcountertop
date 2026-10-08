package tessera

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func screen() *Screen { return kept("") }

func kept(path string) *Screen {
	return New(Options{
		Keep:     path,
		Board:    func() string { return "jc8012p4a1" },
		Size:     func() (int, int) { return 1280, 800 },
		Grid:     func() schema.TesseraGrid { return schema.TesseraGrid{Columns: 4, Rows: 3} },
		Node:     func() string { return "blueberry" },
		Speaking: func() bool { return true },
	})
}

type sender struct {
	t       *testing.T
	s       *Screen
	session string
	seq     int
}

func (c *sender) send(m map[string]any) map[string]any {
	c.t.Helper()
	m["v"] = 2
	if m["op"] != "hello" {
		c.seq++
		m["session"], m["seq"], m["rev"] = c.session, c.seq, "00000000000000aa"
	}
	b, err := json.Marshal(m)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.s.Receive(string(b))
}

func TestALayoutArrivesWholeBeforeItApplies(t *testing.T) {
	c := &sender{t: t, s: screen()}
	hello := c.send(map[string]any{"op": "hello", "request": "34f13e8fb3066a17", "empty": true})
	status, _ := hello["status"].(string)
	if !strings.HasPrefix(status, "Session:") || hello["request"] != "34f13e8fb3066a17" ||
		hello["protocol"] != protocol || status != "Session:"+hello["session"].(string) {
		t.Fatalf("hello answered %v", hello)
	}
	c.session = hello["session"].(string)

	steps := []struct {
		m    map[string]any
		want string
	}{
		{
			map[string]any{
				"op":        "begin",
				"tiles":     1,
				"pages":     1,
				"home":      0,
				"title":     "Living",
				"keepalive": 60,
			},
			"Loading tiles",
		},
		{
			map[string]any{
				"op":           "page",
				"p":            0,
				"id":           "00000000000000b1",
				"title":        "",
				"home_control": false,
				"excluded":     false,
				"items":        []any{},
			},
			"Loading tiles",
		},
		{
			map[string]any{
				"op":     "tile",
				"i":      0,
				"entity": "light.lamp",
				"slot":   0,
				"o":      map[string]any{"size": "single"},
				"a":      map[string]any{},
				"state":  "on",
				"name":   "Lamp",
			},
			"Loading tiles",
		},
		{map[string]any{"op": "ping"}, "Resend needed"},
		{map[string]any{"op": "commit"}, "Synced"},
		{
			map[string]any{
				"op":     "state",
				"i":      0,
				"entity": "light.lamp",
				"a":      map[string]any{},
				"state":  "off",
				"name":   "Lamp",
			},
			"Synced",
		},
	}
	for _, step := range steps {
		if got := c.send(step.m)["status"]; got != step.want {
			t.Fatalf("%v answered %v, want %s", step.m["op"], got, step.want)
		}
	}
	l, applied := c.s.Layout()
	if !applied || len(l.Tiles) != 1 || l.Tiles[0].State != "off" || l.Title != "Living" {
		t.Errorf("layout %+v applied=%v", l, applied)
	}
}

func TestAnOldSessionIsRefused(t *testing.T) {
	c := &sender{t: t, s: screen(), session: "0000000000000001"}
	if got := c.send(map[string]any{"op": "ping"})["status"]; got != "Error: obsolete message" {
		t.Errorf("ping without a session answered %v", got)
	}
}

func TestARepeatedPacketIsADuplicate(t *testing.T) {
	s := screen()
	hello := s.Receive(`{"v":2,"op":"hello","request":"34f13e8fb3066a17"}`)
	packet := `{"v":2,"op":"begin","session":"` + hello["session"].(string) +
		`","seq":1,"rev":"00000000000000aa","tiles":0,"pages":1,"home":0,"title":"","keepalive":60}`
	if got := s.Receive(packet)["status"]; got != "Loading tiles" {
		t.Fatalf("begin answered %v", got)
	}
	if got := s.Receive(packet)["status"]; got != "Loading tiles" {
		t.Errorf("the same begin again answered %v", got)
	}
}

func TestAQuietScreenListsNothing(t *testing.T) {
	s := New(Options{
		Board:    func() string { return "" },
		Size:     func() (int, int) { return 800, 480 },
		Grid:     func() schema.TesseraGrid { return schema.TesseraGrid{Columns: 4, Rows: 4} },
		Node:     func() string { return "show" },
		Speaking: func() bool { return false },
	})
	if len(s.Entities()) != 0 || len(s.Actions()) != 0 {
		t.Error("a screen in native mode still offers Tessera entities")
	}
}

func TestAnAppliedLayoutSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tessera.json")
	c := &sender{t: t, s: kept(path)}
	c.session = c.send(map[string]any{"op": "hello", "request": "34f13e8fb3066a17"})["session"].(string)
	for _, m := range []map[string]any{
		{"op": "begin", "tiles": 1, "pages": 1, "home": 0, "title": "Living", "keepalive": 60},
		{"op": "page", "p": 0, "id": "00000000000000b1", "title": "Lights", "home_control": true, "excluded": false,
			"items": []any{map[string]any{"k": "clock"}}},
		{"op": "tile", "i": 0, "entity": "light.lamp", "slot": 0, "o": map[string]any{"size": "single"}, "a": map[string]any{}, "state": "on", "name": "Lamp"},
		{"op": "commit"},
	} {
		c.send(m)
	}
	l, shown := kept(path).Layout()
	if !shown || l.Title != "Living" || len(l.Tiles) != 1 || l.Pages[0].Bar[0].Kind != "clock" {
		t.Errorf("after a restart %+v shown=%v", l, shown)
	}
}

func TestTheFeedLapsesAndEventsCarryTheInbox(t *testing.T) {
	c := &sender{t: t, s: screen()}
	c.session = c.send(map[string]any{"op": "hello", "request": "34f13e8fb3066a17"})["session"].(string)
	var told []component.Event
	c.s.Told.Listen(func(e component.Event) { told = append(told, e) })
	c.s.Tell("screen_setting", map[string]string{"key": "brightness", "value": "40"})
	if len(told) != 0 {
		t.Fatalf("told %v before an inbox arrived", told)
	}
	for _, m := range []map[string]any{
		{"op": "begin", "tiles": 0, "pages": 1, "home": 0, "title": "", "keepalive": 60,
			"inbox": "kitchen", "numbers": "comma", "percent_space": true},
		{"op": "page", "p": 0, "id": "00000000000000b1", "title": "", "home_control": false, "excluded": false, "items": []any{}},
		{"op": "commit"},
	} {
		c.send(m)
	}
	now := time.Now()
	if !c.s.Fresh(now) || c.s.Fresh(now.Add(3*time.Minute)) {
		t.Errorf(
			"fresh now %v, after three minutes %v",
			c.s.Fresh(now),
			c.s.Fresh(now.Add(3*time.Minute)),
		)
	}
	if l, _ := c.s.Layout(); l.Numbers != (Numbers{Style: 2, Percent: 2}) {
		t.Errorf("numbers %+v", l.Numbers)
	}
	c.s.Tell("screen_setting", map[string]string{"key": "brightness", "value": "40"})
	if len(told) != 1 {
		t.Fatalf("told %+v", told)
	}
	got := told[0]
	if got.Name != "esphome.screen_setting" || got.Data["inbox"] != "kitchen" {
		t.Errorf("told %+v", got)
	}
	if got.Data["session"] != c.session || got.Data["key"] != "brightness" {
		t.Errorf("told %+v", got)
	}
}

func TestASaverWordIsNotATileKey(t *testing.T) {
	c := &sender{t: t, s: screen()}
	c.session = c.send(map[string]any{"op": "hello", "request": "34f13e8fb3066a17"})["session"].(string)
	for _, m := range []map[string]any{
		{"op": "begin", "tiles": 0, "pages": 1, "home": 0, "title": "", "keepalive": 60},
		{"op": "page", "p": 0, "id": "00000000000000b1", "title": "", "home_control": false, "excluded": false, "items": []any{}},
		{"op": "commit"},
	} {
		c.send(m)
	}
	if got := c.send(map[string]any{"op": "saver", "k": ""})["status"]; got != "Synced" {
		t.Errorf("saver answered %v", got)
	}
}
