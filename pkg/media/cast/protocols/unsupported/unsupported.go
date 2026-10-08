package unsupported

import (
	"context"

	"github.com/ygelfand/libcountertop/pkg/media/cast"
	"github.com/ygelfand/libcountertop/pkg/media/playback"
)

type Protocol struct{ out playback.Output }

func init() {
	cast.Define(
		cast.Unsupported,
		func(env cast.Env) cast.Protocol { return &Protocol{out: env.Output} },
	)
}

func (p *Protocol) Name() string         { return cast.Unsupported }
func (p *Protocol) Namespaces() []string { return nil }
func (p *Protocol) Run(context.Context)  {}

func (p *Protocol) Receive(*cast.Application, cast.Message) ([]cast.Message, error) {
	return nil, cast.ErrUnspoken
}

func (p *Protocol) Started(app cast.App) {
	p.out.Attend(&playback.Session{Label: app.Name, Logo: app.Icon, Pictured: app.SupportsVideo})
	p.out.Failed(app.Refused())
}

func (p *Protocol) Ended(cast.App) { p.out.Attend(nil) }
