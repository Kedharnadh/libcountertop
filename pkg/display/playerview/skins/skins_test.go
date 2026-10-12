package skins

import (
	"bytes"
	"testing"

	"golang.org/x/image/webp"
)

func TestEverySkinDecodesWithRegionsInside(t *testing.T) {
	for _, s := range All() {
		img, err := webp.Decode(bytes.NewReader(s.Art()))
		if err != nil {
			t.Errorf("%s: decoding the art: %v", s.Name, err)
			continue
		}
		if img.Bounds().Dx() == 0 {
			t.Errorf("%s: the art is empty", s.Name)
		}
		check := func(what string, r Rect) {
			if r.X0 < 0 || r.Y0 < 0 || r.X1 > 1 || r.Y1 > 1 || r.X0 >= r.X1 || r.Y0 >= r.Y1 {
				t.Errorf("%s: %s %+v is outside the image or empty", s.Name, what, r)
			}
		}
		check("screen", s.Screen)
		for _, k := range s.Keys {
			check("key", k.At)
		}
	}
}

func TestFindKnowsEverySkinAndNothingElse(t *testing.T) {
	for _, s := range All() {
		if got, ok := Find(s.Name); !ok || got.Name != s.Name {
			t.Errorf("Find(%q) = %q, %v", s.Name, got.Name, ok)
		}
	}
	if _, ok := Find("plain"); ok {
		t.Error(`Find("plain") found a skin`)
	}
}
