package setting

import (
	"slices"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"
)

type flavor string

func (f flavor) Label() string { return string(f) }

type order struct{ Flavor flavor }

func TestExtraOptionsAreOfferedTakenAndRefreshed(t *testing.T) {
	extra := []Option{{Value: "x:mint", Label: "Mint"}}
	s := Pick(
		Setting[order]{Name: "flavor", Group: "g", More: func() []Option { return extra }},
		[]flavor{"plain"},
		func(o *order) *flavor { return &o.Flavor },
	)

	var o order
	if err := s.Write(&o, "Mint"); err != nil || o.Flavor != "x:mint" {
		t.Fatalf("writing an extra option gave %q, %v", o.Flavor, err)
	}
	if got := s.LabelOf("x:mint"); got != "Mint" {
		t.Errorf("label %q", got)
	}
	if err := s.Write(&o, "chocolate"); err == nil {
		t.Error("a value nobody offered was taken")
	}

	saved := o
	c := &Controls[order]{
		Table: NewTable("t", []Group{"g"}, []Setting[order]{s}, Messages{}),
		Read:  func() order { return saved },
		Save:  func(Setting[order], string) error { return nil },
	}
	sel := c.Entities()[0].(*esphome.Select)
	extra = append(extra, Option{Value: "x:lime", Label: "Lime"})
	c.Refresh()
	if !slices.Contains(sel.Options, "Lime") || sel.Get() != "Mint" {
		t.Errorf("after refresh the select offers %v and shows %q", sel.Options, sel.Get())
	}
}
