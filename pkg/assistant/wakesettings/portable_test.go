package wakesettings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func portable(t *testing.T) (*Settings, *memory) {
	t.Helper()
	m := &memory{}
	s := New(Options{
		Slots: 2,
		Read:  m.Read,
		Write: func(slot int) schema.WakeWriter { return schema.NewWakeWriter(m, slot) },
	})
	return s, m
}

func TestExportCarriesEachSlotUnderItsNumber(t *testing.T) {
	s, m := portable(t)
	m.wake.Words = []schema.WakeWord{schema.DefaultWakeWord(), schema.DefaultWakeWord()}
	m.wake.Words[1].ID = "hey_jarvis"
	m.wake.Words[1].Threshold = 0.7

	got := s.Controls().Export()
	if got["2.threshold"] != "70" {
		t.Errorf("slot 2 exported threshold %q, want 70", got["2.threshold"])
	}
	if _, ok := got["2.word"]; ok {
		t.Error("the wake word was exported; Home Assistant owns it")
	}
	if got["1.threshold"] != "85" || got["1.delivery"] != string(schema.DefaultDelivery) {
		t.Errorf("slot 1 exported as %v", got)
	}
	if len(m.wake.Words) != 2 || m.wake.Words[0].ID != "" {
		t.Errorf("exporting changed the saved words: %+v", m.wake.Words)
	}
}

func TestImportWritesOnlyItsSlotAndRefreshesTheEntities(t *testing.T) {
	s, m := portable(t)
	changed, err := s.Controls().Import(map[string]string{
		"2.threshold": "60",
		"2.delivery":  string(schema.DeliveryStream),
		"2.follow_up": "5",
	})
	if err != nil || changed != 3 {
		t.Fatalf("changed %d, err %v; want 3", changed, err)
	}
	got := m.wake.Slot(1)
	if got.Threshold != 0.6 || got.Delivery != schema.DeliveryStream || got.FollowUp != 5 {
		t.Errorf("slot 2 saved as %+v", got)
	}
	if m.wake.Slot(0) != schema.DefaultWakeWord() {
		t.Errorf("slot 1 changed: %+v", m.wake.Slot(0))
	}
	if v := s.slots[1].threshold.Get(); v != 0.6 {
		t.Errorf("the sensitivity entity shows %v, want 0.6", v)
	}
}

func TestImportRefusesOutOfRangeValues(t *testing.T) {
	s, m := portable(t)
	if _, err := s.Controls().
		Import(map[string]string{"1.threshold": "20", "1.max_listen": "1"}); err == nil {
		t.Error("out of range values were accepted")
	}
	if m.wake.Slot(0) != schema.DefaultWakeWord() {
		t.Errorf("a refused value was saved: %+v", m.wake.Slot(0))
	}
}

func TestRowsAreTitledByFieldAndGroupedBySlot(t *testing.T) {
	s, _ := portable(t)
	for _, f := range s.Controls().Fields() {
		if f.Name == "2.tone" {
			if f.Title != "Wake sound" || f.Group != "Assistant 2" {
				t.Errorf("2.tone titled %q in %q", f.Title, f.Group)
			}
			return
		}
	}
	t.Error("no 2.tone field")
}
