package volume

import (
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

// How long the card stays up with nobody touching it.
//
// Opened out it stays longer: a button moved a level and the card said so, where opening it out is
// somebody asking to see the rest and then reaching for one of them.
const (
	linger = 5 * time.Second
	opened = 8 * time.Second
)

// card is the volume control over whatever is underneath, and the timer that takes it away.
//
// A view on the shell rather than a claim of its own, because it is something to touch: the shell
// is what turns a contact into a tap on a button or a pull on the bar.
//
// Its lock is taken before the shell's and never the other way round. Nothing the shell calls here
// is called while it holds its own.
type Card struct {
	shell      *shell.Shell
	generation uint64
	mu         sync.Mutex
	held       *shell.Hold
	timer      *time.Timer
	open       bool

	// stream is the one in the circle, which is the one the bar moves.
	stream config.Stream
}

// An overlay: the card hangs off one edge and whatever is underneath carries on showing beside it.
func (c *Card) Covers() bool { return false }

// Selected is the stream in the circle, and whether the card is up at all. What the volume buttons
// ask, so that pressing one moves the level the card is showing rather than a different one.
func (c *Card) Selected() (config.Stream, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.held.Held() {
		return "", false
	}
	return c.stream, true
}

// show puts the card up for a stream's level, or brings the card round to that stream when it is
// already up.
//
// Selecting what moved is the whole point: the circle says which level is on the bar, so a button
// press that changed the alerts level must not leave the card showing the media one.
func (c *Card) Show(stream config.Stream) {
	c.mu.Lock()
	c.stream = stream

	if !c.held.Held() {
		c.held = c.shell.Hold(c)
		c.mu.Unlock()

		c.Linger()
		return
	}
	c.mu.Unlock()

	c.shell.Redraw()
	c.Linger()
}

// Stir shows a new level on a card that is already up, and keeps it up.
//
// It has to repaint. The card is itself what Top returns once it is held, so every change after the
// first comes through here rather than through show — and a Stir that only reset the timer left the
// card displaying the level it opened at while the volume moved underneath it. A press on the
// device hid that, because Adjust repaints the screen itself afterwards and a phone does not.
//
// Nothing happens when the card is not up: that is the levels page being on top, which outranks the
// card and repaints itself.
func (c *Card) Stir() {
	c.mu.Lock()
	up := c.held.Held()
	c.mu.Unlock()

	c.stirred(up)
}

// stirred is Stir once it knows whether the card is up, split out so the decision can be tested
// without a shell and a panel behind it.
func (c *Card) stirred(up bool) {
	if !up {
		return
	}

	c.shell.Redraw()
	c.Linger()
}

// Linger restarts the countdown, which every change and every touch does. Holding a button keeps the
// card up rather than flickering it away between presses.
func (c *Card) Linger() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.generation++
	stay := linger
	if c.open {
		stay = opened
	}

	if c.timer == nil {
		generation := c.generation
		c.timer = time.AfterFunc(stay, func() { c.expire(generation) })
		return
	}
	c.timer.Stop()
	generation := c.generation
	c.timer = time.AfterFunc(stay, func() { c.expire(generation) })
}

// Dismiss takes the card away and folds it back, so the next level that moves brings up the circle
// alone rather than the whole list.
func (c *Card) Dismiss() {
	c.mu.Lock()
	held := c.held
	c.generation++
	c.held, c.open = nil, false

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()

	held.Release()
}

// Expand shows the streams the bar is not moving, or folds them away again.
func (c *Card) Expand() {
	c.mu.Lock()
	c.open = !c.open
	c.mu.Unlock()

	c.Linger()
}

// selects moves the bar to another stream and folds the list away, which is what picking one out
// of a list means: the thing picked becomes the thing shown.
func (c *Card) selects(stream config.Stream) func() {
	return func() {
		c.mu.Lock()
		c.stream, c.open = stream, false
		c.mu.Unlock()

		c.Linger()
	}
}

// Shows implements the test Set makes before putting the card up. The card already shows whichever
// stream is Selected, so a change to that one is something to redraw rather than announce over the
// top; a change to another still brings the card round to it.
func (c *Card) Shows(s config.Stream) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.held.Held() && c.stream == s
}

func NewCard(stack *shell.Shell) *Card { return &Card{shell: stack} }
func (c *Card) Picked() (config.Stream, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stream, c.open
}
func (c *Card) Pick(stream config.Stream) { c.selects(stream)() }
func (c *Card) Close()                    { c.Dismiss() }
func (c *Card) expire(generation uint64) {
	c.mu.Lock()
	if c.generation != generation {
		c.mu.Unlock()
		return
	}
	held := c.held
	c.held, c.open, c.timer = nil, false, nil
	c.mu.Unlock()
	held.Release()
}
