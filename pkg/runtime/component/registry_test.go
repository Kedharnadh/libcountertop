package component

import (
	"context"
	"slices"
	"testing"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"
)

type testConfig struct{}

type bare struct{ name string }

func (b bare) Name() string { return b.name }

type full struct {
	bare
	entity   *esphome.Switch
	restored int
	ran      bool
	running  chan struct{}
}

func newFull(name string) *full {
	return &full{
		bare:    bare{name: name},
		running: make(chan struct{}),
		entity:  &esphome.Switch{Base: esphome.Base{ObjectID: name}},
	}
}

func (f *full) Entities() []esphome.Entity { return []esphome.Entity{f.entity} }
func (f *full) Restore(c testConfig)       { f.restored++ }
func (f *full) Run(context.Context) error  { f.ran = true; close(f.running); return nil }

func names(cs []Component) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name())
	}
	return out
}

func val(c Component) func() Component { return func() Component { return c } }

func TestOrderIsDeclaredNotRegistered(t *testing.T) {
	r := New[testConfig](nil)

	r.Add(Network, val(bare{"api"}))
	r.Add(Hardware, val(bare{"speaker"}), Order(20))
	r.Add(Device, val(bare{"voice"}))
	r.Add(Hardware, val(bare{"led"}), Order(10))
	r.Add(Hardware, val(bare{"zzz"}), Order(10))
	r.Add(Hardware, val(bare{"aaa"}), Order(10))

	want := []string{"aaa", "led", "zzz", "speaker", "voice", "api"}
	if got := names(r.All()); !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

type once struct {
	bare
	started chan struct{}
	closed  bool
}

func newOnce(name string) *once {
	return &once{bare: bare{name: name}, started: make(chan struct{})}
}

func (o *once) Start(context.Context) error { close(o.started); return nil }
func (o *once) Close() error                { o.closed = true; return nil }

func TestWhatReachesTheSupervisor(t *testing.T) {
	r := New[testConfig](nil)
	loop, setup := newFull("speaker"), newOnce("procs")

	r.Add(Hardware, val(bare{"nothing"}))
	r.Add(Hardware, val(loop))
	r.Add(Hardware, val(setup))

	g := r.Group()
	if n := len(g.Status()); n != 2 {
		t.Fatalf("the group took %d services, want the loop and the one-shot", n)
	}

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- g.Run(ctx) }()

	select {
	case <-setup.started:
	case <-time.After(5 * time.Second):
		t.Fatal("a component with no loop was never started")
	}

	select {
	case <-loop.running:
	case <-time.After(5 * time.Second):
		t.Fatal("the component loop never started")
	}
	stop()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !loop.ran {
		t.Error("the component with a loop never ran")
	}
	if !setup.closed {
		t.Error("a one-shot component was not closed on the way out")
	}
}

func TestOptionalHalves(t *testing.T) {
	r := New[testConfig](nil)
	second, first := newFull("second"), newFull("first")

	r.Add(Device, val(second), Order(2))
	r.Add(Device, val(first), Order(1))
	r.Add(Device, val(bare{"nothing"}), Order(3))

	ents := r.Entities()
	if len(ents) != 2 {
		t.Fatalf("collected %d entities, want 2", len(ents))
	}
	if got := ents[0].(*esphome.Switch).ObjectID; got != "first" {
		t.Errorf("entities led with %q", got)
	}

	r.Restore(testConfig{})

	if first.restored != 1 || second.restored != 1 {
		t.Errorf("restored %d and %d times, want once each", first.restored, second.restored)
	}
}

func TestEmptyRegistry(t *testing.T) {
	r := New[testConfig](nil)

	if got := r.All(); len(got) != 0 {
		t.Errorf("All = %v", got)
	}
	if got := r.Entities(); got != nil {
		t.Errorf("Entities = %v", got)
	}
	if got := r.Handlers(); got != nil {
		t.Errorf("Handlers = %v", got)
	}
	r.Restore(testConfig{})
}

func TestRegisterTakesATypedMaker(t *testing.T) {
	r := New[testConfig](nil)
	Register(r, Device, func() bare { return bare{name: "typed"} }, Order(1))
	all := r.All()
	if len(all) != 1 || all[0].Name() != "typed" {
		t.Fatalf("registered %v, want the typed component", all)
	}
}
