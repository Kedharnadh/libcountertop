package wpa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The events worth acting on. wpa_supplicant sends many more.
const (
	EventConnected    = "CTRL-EVENT-CONNECTED"
	EventStateChange  = "CTRL-EVENT-STATE-CHANGE"
	EventDisconnected = "CTRL-EVENT-DISCONNECTED"
	EventTerminating  = "CTRL-EVENT-TERMINATING"
)

// stateCompleted is wpa_state COMPLETED, as the state change reports it by number.
const stateCompleted = 9

// StateChange is what a CTRL-EVENT-STATE-CHANGE carries: everything STATUS would have answered about
// the association, without asking.
//
// Asking is what caused the trouble. Answering STATUS makes the supplicant look its own address up
// and pass back through set_state, which emits this event again — so a STATUS sent from a handler
// for it is a loop that runs until something else stops it.
type StateChange struct {
	Completed bool
	SSID      string
	BSSID     string
}

// ParseState reads the fields off a state change. The text is "id=0 state=9 BSSID=.. SSID=..", and an
// SSID with a space in it is the rest of the line.
func ParseState(text string) (StateChange, bool) {
	rest, ok := strings.CutPrefix(text, EventStateChange+" ")
	if !ok {
		return StateChange{}, false
	}

	var s StateChange
	hasState := false
	for rest != "" {
		var field string
		field, rest, _ = strings.Cut(rest, " ")

		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}

		switch key {
		case "state":
			n, err := strconv.Atoi(value)
			if err != nil {
				return StateChange{}, false
			}
			hasState = true
			s.Completed = n == stateCompleted
		case "BSSID":
			s.BSSID = value
		case "SSID":
			// Last, and may hold spaces, so it is the remainder rather than the field.
			s.SSID = value
			if rest != "" {
				s.SSID += " " + rest
			}
			rest = ""
		}
	}
	return s, hasState
}

// Event is an unsolicited message from the supplicant: a priority, and the text after it.
type Event struct {
	Priority int
	Text     string
}

// Is reports whether the event is of a kind, which is the first word of the text.
func (e Event) Is(kind string) bool { return e.Text == kind || strings.HasPrefix(e.Text, kind+" ") }

// EventConn is a dedicated event connection; command replies must not share another reader.
type EventConn interface {
	Cmd(string) (string, error)
	Read([]byte) (int, error)
	SetReadDeadline(time.Time) error
}

// Monitor attaches before taking the initial snapshot so association changes cannot fall in a
// gap between STATUS and subscribing. The snapshot uses a separate command connection.
func Monitor(ctx context.Context, c EventConn, initial func(), handler func(Event)) error {
	if out, err := c.Cmd("ATTACH"); err != nil {
		return err
	} else if strings.TrimSpace(out) != "OK" {
		return fmt.Errorf("wifi: ATTACH said %q", out)
	}
	defer c.Cmd("DETACH")
	if initial != nil {
		initial()
	}

	buf := make([]byte, 4096)
	for {
		if ctx.Err() != nil {
			return nil
		}

		// A deadline rather than a blocking read, so canceling does not wait for an event that
		// may never come.
		if err := c.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}

		n, err := c.Read(buf)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return fmt.Errorf("wifi: reading events: %w", err)
		}
		if e, ok := ParseEvent(string(buf[:n])); ok {
			handler(e)
		}
	}
}

// ParseEvent reads the "<3>CTRL-EVENT-CONNECTED ..." form.
func ParseEvent(raw string) (Event, bool) {
	raw = strings.TrimRight(raw, "\n")
	if !strings.HasPrefix(raw, "<") {
		return Event{}, false
	}
	end := strings.Index(raw, ">")
	if end < 0 {
		return Event{}, false
	}

	priority, err := strconv.Atoi(raw[1:end])
	if err != nil {
		return Event{}, false
	}
	return Event{Priority: priority, Text: raw[end+1:]}, true
}

func isTimeout(err error) bool {
	type timeout interface{ Timeout() bool }
	t, ok := err.(timeout)
	return ok && t.Timeout()
}
