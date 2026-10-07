package policy

import (
	"errors"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/fetch"
)

func TestSaveFailureAppliesActiveConfiguration(t *testing.T) {
	defer fetch.Verify(true)
	active := true
	committed := false
	p := New(Options{
		Read: func() bool { return active },
		Save: func(v bool) error {
			if committed {
				active = v
			}
			return errors.New("save failure")
		},
	})
	p.Restore(true)
	p.SetVerify(false)
	if !p.Verifying() || !p.verify.Get() {
		t.Fatal("failed write disabled verification")
	}
	committed = true
	p.SetVerify(false)
	if p.Verifying() || p.verify.Get() {
		t.Fatal("post-rename error hid active configuration")
	}
}
