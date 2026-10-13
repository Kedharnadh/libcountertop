package policy

import (
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type (
	Setting = setting.Setting[schema.Network]
	Table   = setting.Table[schema.Network]
)

const Group setting.Group = "Network"

func NewTable() *Table {
	return setting.NewTable("network", []setting.Group{Group}, []Setting{
		setting.Flag(Setting{
			Name: "verify", Group: Group, ID: "verify_certificates", Icon: "mdi:certificate",
		}, func(c *schema.Network) *bool { return &c.Verify }),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

type Policy struct {
	settings storage.Section[schema.Network]
	knobs    *setting.Controls[schema.Network]
}

func New(settings storage.Section[schema.Network]) *Policy {
	p := &Policy{settings: settings}
	p.knobs = &setting.Controls[schema.Network]{
		Table: NewTable(),
		Read:  settings.Read,
		Save:  p.save,
	}
	return p
}

func (p *Policy) Name() string                                { return "network" }
func (p *Policy) Entities() []esphome.Entity                  { return p.knobs.Entities() }
func (p *Policy) Settings() *setting.Controls[schema.Network] { return p.knobs }
func (p *Policy) Restore(verify bool)                         { p.apply(verify) }
func (p *Policy) Verifying() bool                             { return !fetch.Skipping() }

func (p *Policy) SetVerify(verify bool) {
	if err := p.save(p.knobs.Table.MustRow("verify"), setting.OnOff(verify)); err != nil {
		slog.Error("saving certificate policy failed", "err", err)
	}
}

func (p *Policy) save(s Setting, v string) error {
	err := setting.Store(p.settings, s, v)
	p.apply(p.settings.Read().Verify)
	return err
}

func (p *Policy) apply(verify bool) {
	fetch.Verify(verify)
	at := p.settings.Read()
	at.Verify = verify
	p.knobs.PublishFrom(at)
	if !verify {
		slog.Warn("certificates are not being verified")
	}
}
