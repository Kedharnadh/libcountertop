package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestCanceledGestureReleasesItsContact(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var contacts []Contact
	input := NewInput(InputOptions{Step: time.Millisecond, Deliver: func(c Contact) {
		contacts = append(contacts, c)
		if c.Phase == 0 {
			cancel()
		}
	}})
	factory := func() *cobra.Command {
		root := NewRoot()
		root.AddCommand(input.Commands()...)
		Organize(root)
		return root
	}
	_, err := Execute(ctx, factory, []string{"input", "hold", "10", "20", "60000"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("hold error=%v", err)
	}
	if len(contacts) != 2 || contacts[0].Phase != 0 || contacts[1].Phase != 2 || contacts[0].ID != contacts[1].ID || contacts[0].ID >= 0 {
		t.Fatalf("canceled contact sequence=%+v", contacts)
	}
}

func TestSwipeUsesBoardCadenceAndViewedCoordinates(t *testing.T) {
	var contacts []Contact
	input := NewInput(InputOptions{Step: time.Microsecond, Steps: 2, Deliver: func(c Contact) { contacts = append(contacts, c) }})
	factory := func() *cobra.Command {
		root := NewRoot()
		root.AddCommand(input.Commands()...)
		Organize(root)
		return root
	}
	if _, err := Execute(t.Context(), factory, []string{"input", "swipe", "0", "10", "20", "30"}); err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 4 || contacts[1].X != 10 || contacts[1].Y != 20 || contacts[2].X != 20 || contacts[3].Phase != 2 {
		t.Fatalf("swipe=%+v", contacts)
	}
}
