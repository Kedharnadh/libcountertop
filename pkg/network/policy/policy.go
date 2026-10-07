// Package policy binds the common TLS setting to the shared HTTP transport.
package policy

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Policy struct {
	settings storage.Section[schema.Network]
	verify   *esphome.Switch
}

func New(settings storage.Section[schema.Network]) *Policy {
	p := &Policy{settings: settings, verify: &esphome.Switch{Base: esphome.Base{ObjectID: "verify_certificates", Name: "Verify certificates", Icon: "mdi:certificate", Category: esphome.CategoryConfig}}}
	p.verify.OnCommand = p.SetVerify
	return p
}

func (p *Policy) Name() string               { return "network" }
func (p *Policy) Entities() []esphome.Entity { return []esphome.Entity{p.verify} }
func (p *Policy) Restore(verify bool)        { p.apply(verify) }
func (p *Policy) Verifying() bool            { return !fetch.Skipping() }
func (p *Policy) SetVerify(verify bool) {
	err := p.settings.Update(func(c *schema.Network) { c.Verify = verify })
	p.apply(p.settings.Read().Verify)
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
