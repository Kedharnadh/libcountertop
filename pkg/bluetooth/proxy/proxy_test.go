package proxy

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/bluetooth/hci"
)

func TestQueueNeverWaitsForHomeAssistant(t *testing.T) {
	b := &Proxy{reports: make(chan hci.Advertisement, 2)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			b.queue(hci.Advertisement{})
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queueing blocked with nobody reading, which would stall the controller")
	}

	if got := b.Dropped(); got != 98 {
		t.Errorf("%d reports dropped past a queue of 2, want 98", got)
	}
}

func TestQueueKeepsWhatFits(t *testing.T) {
	b := &Proxy{reports: make(chan hci.Advertisement, 4)}

	for i := range 4 {
		b.queue(hci.Advertisement{RSSI: int8(-i)})
	}

	for i := range 4 {
		select {
		case a := <-b.reports:
			if a.RSSI != int8(-i) {
				t.Errorf("report %d has rssi %d, want %d", i, a.RSSI, -i)
			}
		default:
			t.Fatalf("only %d reports queued, want 4", i)
		}
	}

	if got := b.Dropped(); got != 0 {
		t.Errorf("%d dropped with room for every report, want 0", got)
	}
}
