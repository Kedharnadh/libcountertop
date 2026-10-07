// Package policy binds the common TLS setting to the shared HTTP transport.
package policy

import (
	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/libcountertop/pkg/fetch"
	"log/slog"
)

type Options struct {
	Read func() bool
	Save func(bool) error
}

type Policy struct {
	options Options
	verify  *esphome.Switch
}

func New(options Options) *Policy {
	p := &Policy{options: options, verify: &esphome.Switch{Base: esphome.Base{ObjectID: "verify_certificates", Name: "Verify certificates", Icon: "mdi:certificate", Category: esphome.CategoryConfig}}}
	p.verify.OnCommand = p.SetVerify
	return p
}

func (p *Policy) Name() string               { return "network" }
func (p *Policy) Entities() []esphome.Entity { return []esphome.Entity{p.verify} }
func (p *Policy) Restore(verify bool)        { p.apply(verify) }
func (p *Policy) Verifying() bool            { return !fetch.Skipping() }
func (p *Policy) SetVerify(verify bool) {
	err := p.options.Save(verify)
	p.apply(p.options.Read())
	if err != nil {
		slog.Error("saving certificate policy failed", "err", err)
	}
}
func (p *Policy) apply(verify bool) {
	fetch.Verify(verify)
	p.verify.Set(verify)
	if !verify {
		slog.Warn("certificates are not being verified")
	}
}
