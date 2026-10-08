package nativeapi

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

// Home Assistant drops the connection after 4.5 keepalive intervals.
const writeTimeout = 20 * time.Second

type Registry interface {
	Entities() []esphome.Entity
	Actions() []*esphome.Action
	Handlers() []esphome.Handler
}

type Adoption interface {
	Adopted() bool
	SetAdopted(on bool) error
}

type Identity struct {
	Addr     string
	Platform string
	Board    string
	Info     esphome.Info
}

type Dependencies struct {
	KeyPath      string
	Identity     func() (Identity, error)
	Bluetooth    func() esphome.BluetoothFeature
	Registry     Registry
	Adoption     Adoption
	Logs         LogSource
	OnSubscribed func()
}

type Server struct {
	deps Dependencies

	srv *esphome.Server
	id  Identity

	reconnect chan struct{}
	announced sync.Once
	port      atomic.Int32
}

func New(deps Dependencies) *Server {
	return &Server{deps: deps, reconnect: make(chan struct{}, 1)}
}

func (a *Server) Name() string { return "api" }

func (a *Server) Startup() component.Progress {
	port := a.port.Load()
	if port == 0 {
		return component.Progress{Doing: "opening the port"}
	}
	return component.Progress{Done: true, Doing: fmt.Sprintf("port %d", port)}
}

func (a *Server) Start(context.Context) error {
	psk, err := loadPSK(a.deps.KeyPath)
	if err != nil {
		return err
	}
	id, err := a.deps.Identity()
	if err != nil {
		return err
	}
	info := id.Info

	registered := a.deps.Registry.Entities()
	ents := esphome.NewEntities()
	if err := ents.Add(registered...); err != nil {
		return err
	}
	if err := ents.AddActions(a.deps.Registry.Actions()...); err != nil {
		return err
	}

	info.Devices = SubDevices(info.Devices, registered)
	if a.deps.Bluetooth != nil {
		info.BluetoothFeatures = a.deps.Bluetooth()
	}

	a.id = id
	a.srv = &esphome.Server{
		Addr:               id.Addr,
		WriteTimeout:       writeTimeout,
		Info:               info,
		PSK:                psk,
		Logger:             slog.Default(),
		OnSetEncryptionKey: func(k esphome.PSK) error { return writePSK(a.deps.KeyPath, k) },
		OnSubscribed:       a.subscribed,
		Handler: esphome.Chain(
			append([]esphome.Handler{ents}, a.deps.Registry.Handlers()...)...),
	}
	return nil
}

func (a *Server) Run(ctx context.Context) error {
	defer a.port.Store(0)
	if a.deps.Logs != nil {
		safe.Go("logs", func() { a.pipeLogs(ctx) })
	}

	for {
		ln, err := net.Listen("tcp", a.srv.Addr)
		if err != nil {
			return fmt.Errorf("api: listen %s: %w", a.srv.Addr, err)
		}
		port := ln.Addr().(*net.TCPAddr).Port
		a.port.Store(int32(port))
		slog.Info("serving", "addr", ln.Addr(), "node", a.srv.Info.Name)

		a.announced.Do(func() {
			safe.Go("mdns", func() { a.advertise(ctx, port) })
		})

		serving, stop := context.WithCancel(ctx)
		go func() {
			select {
			case <-a.reconnect:
			case <-serving.Done():
			}
			stop()
		}()

		err = a.srv.Serve(serving, ln)
		stop()
		a.port.Store(0)

		if err != nil || ctx.Err() != nil {
			return err
		}

		if a.deps.Bluetooth != nil {
			a.srv.Info.BluetoothFeatures = a.deps.Bluetooth()
			slog.Info("serving again", "bluetooth", a.srv.Info.BluetoothFeatures)
		}
	}
}

func (a *Server) Fire(e component.Event) {
	if a.srv == nil {
		return
	}
	if err := a.srv.FireEvent(e.Name, e.Data); err != nil {
		slog.Debug("firing an event failed", "event", e.Name, "err", err)
	}
}

func (a *Server) Reconnect() {
	select {
	case a.reconnect <- struct{}{}:
	default:
	}
}

func (a *Server) Adopted() bool { return a.deps.Adoption.Adopted() }

func (a *Server) SetAdopted(on bool) {
	if err := a.deps.Adoption.SetAdopted(on); err != nil {
		slog.Error("saving a setting failed", "setting", "api.adopted", "err", err)
	}
}

func (a *Server) subscribed() {
	if !a.deps.Adoption.Adopted() {
		if err := a.deps.Adoption.SetAdopted(true); err != nil {
			slog.Error("recording adoption failed", "err", err)
		} else {
			slog.Info("adopted")
		}
	}
	if a.deps.OnSubscribed != nil {
		a.deps.OnSubscribed()
	}
}
