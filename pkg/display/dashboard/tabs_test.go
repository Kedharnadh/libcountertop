package dashboard

import "testing"

func TestTabs(t *testing.T) {
	draws := 0
	var offered []Tab
	strip := NewTabs(func() { draws++ })
	strip.Add(100, func() []Tab { return []Tab{{Kind: "b", Key: "two"}} })
	strip.Add(10, func() []Tab { return offered })

	offered = []Tab{{Kind: "a", Key: "one"}}
	if got := strip.List(); len(got) != 2 || got[0].Key != "one" || got[1].Key != "two" {
		t.Fatalf("tabs %v, want one then two", got)
	}

	strip.Show("one")
	if tab, ok := strip.Showing(); !ok || tab.Kind != "a" {
		t.Errorf("showing %v %v", tab, ok)
	}
	strip.Show("one")
	if _, ok := strip.Showing(); ok {
		t.Error("tapping the open tab again should go back to the clock")
	}

	strip.Show("one")
	offered = nil
	if _, ok := strip.Showing(); ok {
		t.Error("a tab that is no longer offered is still showing")
	}

	strip.Show("two")
	strip.Clock()
	if _, ok := strip.Showing(); ok {
		t.Error("Clock left a tab open")
	}
	before := draws
	strip.Clock()
	if draws != before {
		t.Error("Clock redrew with nothing open")
	}
}
