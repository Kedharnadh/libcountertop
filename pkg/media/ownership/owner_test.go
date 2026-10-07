package ownership

import "testing"

type source struct{ kind uint8 }

func TestStaleSourceCannotReleaseOrEndReplacement(t *testing.T) {
	o := New(func(s *source) uint8 { return s.kind })
	old, next := &source{kind: 1}, &source{kind: 1}
	o.Begin(old)
	o.Begin(next)
	if o.Release(old) {
		t.Fatal("old source released replacement")
	}
	o.End(old)
	if got, held := o.Get(); !held || got != next {
		t.Fatal("replacement lost")
	}
	if _, fresh, _ := o.Begin(next); fresh {
		t.Fatal("old completion ended replacement's session")
	}
	if !o.Release(next) {
		t.Fatal("current source could not release")
	}
}

func TestKindCallbackCanReadOwner(t *testing.T) {
	var o *Owner[*source]
	o = New(func(s *source) uint8 { o.Get(); return s.kind })
	o.Set(&source{kind: 1})
	o.Begin(&source{kind: 2})
}
