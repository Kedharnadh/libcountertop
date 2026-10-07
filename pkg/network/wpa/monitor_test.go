package wpa

import (
	"context"
	"io"
	"slices"
	"testing"
	"time"
)

type eventConn struct {
	attached bool
	frames   []string
	commands []string
}

func (c *eventConn) Cmd(command string) (string, error) {
	c.commands = append(c.commands, command)
	c.attached = command == "ATTACH"
	return "OK", nil
}

func (c *eventConn) Read(buf []byte) (int, error) {
	if len(c.frames) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, c.frames[0])
	c.frames = c.frames[1:]
	return n, nil
}

func (*eventConn) SetReadDeadline(time.Time) error { return nil }

func TestMonitorSubscribesBeforeSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &eventConn{}
	var got Event
	err := Monitor(ctx, c, func() {
		if !c.attached {
			t.Fatal("initial snapshot ran before subscribing")
		}
		// An association completes while the initial STATUS request is in flight.
		// Only an attached listener receives this event.
		c.frames = append(c.frames, "<3>CTRL-EVENT-CONNECTED - Connection completed")
	}, func(e Event) {
		got = e
		cancel()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Is(EventConnected) {
		t.Fatalf("completion during snapshot was lost: %+v", got)
	}
	if !slices.Equal(c.commands, []string{"ATTACH", "DETACH"}) {
		t.Errorf("monitor commands = %v", c.commands)
	}
}

func TestStateWithoutStateNumberIsNotADisconnect(t *testing.T) {
	if _, ok := ParseState("CTRL-EVENT-STATE-CHANGE id=0 SSID=home"); ok {
		t.Fatal("incomplete vendor state event accepted as a disconnection")
	}
	if (Event{Text: EventConnected + "-OTHER"}).Is(EventConnected) {
		t.Fatal("unrelated event accepted as CONNECTED")
	}
}
