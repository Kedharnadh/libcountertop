package policy

import (
	"errors"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestSaveFailureAppliesActiveConfiguration(t *testing.T) {
	defer fetch.Verify(true)
	active := true
	committed := false
	p := New(policySettings{active: &active, committed: &committed})
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

type policySettings struct{ active, committed *bool }

func (s policySettings) Read() schema.Network { return schema.Network{Verify: *s.active} }
func (s policySettings) Update(change func(*schema.Network)) error {
	c := s.Read()
	change(&c)
	if *s.committed {
		*s.active = c.Verify
	}
	return errors.New("save failure")
}
