package castreceiver

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync/atomic"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/media/cast"
	"github.com/ygelfand/libcountertop/pkg/media/cast/protocols/youtube"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type ReceiverOptions struct {
	Settings storage.Section[schema.Cast]
	DeviceID uint32
	Network  func(ctx context.Context) bool
	Server   Config
}

type Receiver struct {
	settings storage.Section[schema.Cast]
	network  func(ctx context.Context) bool

	enable *esphome.Switch
	demand *esphome.Switch
	skip   *esphome.Text
	delay  *esphome.Number

	server *Server

	boot atomic.Pointer[startup.Progress]

	oracle  *esphome.Text
	source  *esphome.TextSensor
	expires *esphome.Sensor
}

func NewReceiver(opts ReceiverOptions) *Receiver {
	r := &Receiver{settings: opts.Settings, network: opts.Network}
	c := opts.Settings.Read()

	r.enable = &esphome.Switch{
		Base: esphome.Base{
			ObjectID: "cast_receiver",
			Name:     "Cast receiver",
			Icon:     "mdi:cast",
			Category: esphome.CategoryConfig,
		},
	}
	r.oracle = &esphome.Text{
		Base: esphome.Base{
			ObjectID: "cast_oracle",
			Name:     "Cast oracle",
			Icon:     "mdi:certificate",
			Category: esphome.CategoryConfig,
		},
		MaxLength: 255,
	}
	r.source = &esphome.TextSensor{
		Base: esphome.Base{
			ObjectID: "cast_credentials",
			Name:     "Cast credentials",
			Icon:     "mdi:certificate",
			Category: esphome.CategoryDiagnostic,
		},
	}
	r.expires = &esphome.Sensor{
		Base: esphome.Base{
			ObjectID: "cast_credentials_expire",
			Name:     "Cast credentials expire",
			Icon:     "mdi:certificate-outline",
			Category: esphome.CategoryDiagnostic,
		},
		DeviceClass: "timestamp",
	}
	r.demand = &esphome.Switch{
		Base: esphome.Base{
			ObjectID: "youtube_lounge_on_demand",
			Name:     "YouTube lounge on demand",
			Icon:     "mdi:youtube",
			Category: esphome.CategoryConfig,
		},
	}
	r.skip = &esphome.Text{
		Base: esphome.Base{
			ObjectID: "youtube_sponsorblock",
			Name:     "YouTube SponsorBlock categories",
			Icon:     "mdi:debug-step-over",
			Category: esphome.CategoryConfig,
		},
		MaxLength: 255,
	}
	r.delay = &esphome.Number{
		Base: esphome.Base{
			ObjectID: "youtube_live_delay",
			Name:     "YouTube live delay",
			Icon:     "mdi:timer-sand",
			Category: esphome.CategoryConfig,
		},
		Min: schema.LiveDelayLeast, Max: schema.LiveDelayMost, Step: 1, Unit: "s",
		Mode: esphome.NumberSlider,
	}

	for _, b := range []*esphome.Base{
		&r.enable.Base, &r.oracle.Base, &r.source.Base, &r.expires.Base,
		&r.demand.Base, &r.skip.Base, &r.delay.Base,
	} {
		b.DeviceID = opts.DeviceID
	}

	r.demand.Set(c.YouTube.OnDemand)
	r.demand.OnCommand = r.SetLoungeOnDemand
	r.skip.Set(strings.Join(c.YouTube.Skip, ","))
	r.skip.OnCommand = r.SetSkip
	r.delay.Set(float32(c.YouTube.LiveDelay))
	r.delay.OnCommand = func(v float32) { r.SetLiveDelay(int(v)) }
	r.enable.Set(c.Receiver)
	r.enable.OnCommand = r.SetReceiver
	r.oracle.Set(c.Oracle)
	r.oracle.OnCommand = r.SetOracle

	server := opts.Server
	server.Oracle = func() string { return r.settings.Read().Oracle }
	server.Credentials = r.credentials
	r.server = New(server)
	return r
}

func (r *Receiver) write() schema.CastWriter { return schema.NewCastWriter(r.settings) }

func (r *Receiver) Name() string { return "cast receiver" }

func (r *Receiver) Entities() []esphome.Entity {
	return []esphome.Entity{r.enable, r.oracle, r.source, r.expires, r.demand, r.skip, r.delay}
}

func (r *Receiver) SetSkip(text string) {
	categories := youtube.ParseCategories(text)
	if err := r.write().Skip(categories); err != nil {
		slog.Error("saving the sponsorblock categories failed", "err", err)
		return
	}
	r.skip.Set(strings.Join(categories, ","))
}

func (r *Receiver) SetLiveDelay(seconds int) {
	if err := r.write().LiveDelay(seconds); err != nil {
		slog.Error("saving the live delay failed", "err", err)
		return
	}
	r.delay.Set(float32(r.settings.Read().YouTube.LiveDelay))
}

func (r *Receiver) SetLoungeOnDemand(on bool) {
	r.demand.Set(on)
	if err := r.write().LoungeOnDemand(on); err != nil {
		slog.Error("saving the lounge setting failed", "err", err)
		return
	}
	if !r.Enabled() {
		return
	}
	r.server.Down()
	if err := r.server.Up(); err != nil {
		slog.Error("the cast receiver would not start", "err", err)
	}
}

func (r *Receiver) SetOracle(url string) {
	url = strings.TrimSpace(url)
	if err := r.write().Oracle(url); err != nil {
		slog.Error("saving the cast oracle failed", "err", err)
		return
	}
	r.oracle.Set(url)
	r.server.Retarget()
}

func (r *Receiver) credentials(c *cast.Credentials) {
	if !c.Remote {
		r.source.Set("self-signed")
		r.expires.Set(float32(math.NaN()))
		return
	}
	r.source.Set("remote-signed")
	r.expires.Set(float32(c.NotAfter.Unix()))
}

func (r *Receiver) Enabled() bool { return r.settings.Read().Receiver }

func (r *Receiver) Listening() bool { return r.server.Listening() }

func (r *Receiver) Start(context.Context) error {
	if !r.Enabled() {
		r.booted(startup.Progress{Done: true, Doing: "off"})
		return nil
	}
	r.booted(startup.Progress{Doing: "waiting for the network"})
	return nil
}

func (r *Receiver) Run(ctx context.Context) error {
	if !r.Enabled() {
		<-ctx.Done()
		return nil
	}
	keys := make(chan error, 1)
	safe.Go("cast keys", func() { keys <- r.server.Prepare() })
	if r.network(ctx) && r.Enabled() {
		var err error
		select {
		case err = <-keys:
		default:
			r.booted(startup.Progress{Doing: "making keys"})
			err = <-keys
		}
		if err == nil {
			err = r.server.Up()
		}
		if err != nil {
			slog.Error("the cast receiver would not start", "err", err)
			r.booted(startup.Progress{Failed: true, Doing: err.Error()})
		} else {
			r.booted(startup.Progress{Done: true, Doing: fmt.Sprintf("port %d", cast.Port)})
		}
	}
	<-ctx.Done()
	return nil
}

func (r *Receiver) booted(p startup.Progress) { r.boot.Store(&p) }

func (r *Receiver) Startup() startup.Progress {
	if p := r.boot.Load(); p != nil {
		return *p
	}
	return startup.Progress{Doing: "waiting for the network"}
}

func (r *Receiver) Close() error {
	r.server.Down()
	return nil
}

func (r *Receiver) PrimeRegistered() bool { return r.server.Registered() }

func (r *Receiver) ResetPrime() {
	r.server.ForgetRegistrations()
	slog.Info("prime video registration reset")
}

func (r *Receiver) SetPrimeSkipIntro(on bool) {
	if err := r.write().PrimeSkipIntro(on); err != nil {
		slog.Error("saving the prime video skip intro setting failed", "err", err)
	}
}

func (r *Receiver) SetPrimePersist(on bool) {
	if err := r.write().PrimePersist(on); err != nil {
		slog.Error("saving the prime video registration setting failed", "err", err)
		return
	}
	if !on {
		r.server.UnsaveRegistrations()
	}
}

func (r *Receiver) SetReceiver(on bool) {
	r.enable.Set(on)

	if err := r.write().Receiver(on); err != nil {
		slog.Error("saving the cast setting failed", "err", err)
		return
	}

	if !on {
		r.server.Down()
		r.booted(startup.Progress{Done: true, Doing: "off"})
		return
	}
	if err := r.server.Up(); err != nil {
		slog.Error("the cast receiver would not start", "err", err)
		r.booted(startup.Progress{Failed: true, Doing: err.Error()})
		return
	}
	r.booted(startup.Progress{Done: true, Doing: fmt.Sprintf("port %d", cast.Port)})
}

func (r *Receiver) Playing() *cast.Media { return r.server.Playing() }
