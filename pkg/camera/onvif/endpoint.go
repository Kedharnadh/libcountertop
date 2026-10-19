package onvif

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/network/digest"
)

const SnapshotPath = "/onvif/snapshot.jpg"

type Config struct {
	Port     int
	Device   Device
	Hardware string
	Snapshot func() ([]byte, error)
	Auth     digest.Credentials
}

type Endpoint struct {
	http  *http.Server
	found *Responder
}

func Serve(cfg Config) *Endpoint {
	mux := http.NewServeMux()
	svc := &Service{Auth: cfg.Auth, challenge: digest.New(cfg.Hardware)}
	if cfg.Snapshot != nil {
		cfg.Device.SnapshotPath = SnapshotPath
		mux.HandleFunc(SnapshotPath, snapshot(cfg.Snapshot, svc))
	}
	svc.Device = cfg.Device
	mux.Handle(DevicePath, svc)
	mux.Handle(MediaPath, svc)

	e := &Endpoint{http: &http.Server{Handler: mux}}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.Port))
	if err != nil {
		slog.Warn("onvif is not answering", "port", cfg.Port, "err", err)
		return e
	}
	go func() {
		if err := e.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("onvif stopped serving", "port", cfg.Port, "err", err)
		}
	}()

	e.found = &Responder{
		Hardware: cfg.Hardware,
		UUID:     NewUUID(),
		Name:     cfg.Device.Name,
		XAddr: func(ip net.IP) string {
			return "http://" + net.JoinHostPort(ip.String(), strconv.Itoa(cfg.Port)) + DevicePath
		},
	}
	if err := e.found.Listen(); err != nil {
		slog.Warn("onvif discovery is not answering", "err", err)
		e.found = nil
	}
	return e
}

func (e *Endpoint) Close() {
	if e == nil {
		return
	}
	_ = e.http.Close()
	if e.found != nil {
		e.found.Close()
	}
}

func snapshot(still func() ([]byte, error), svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !svc.Auth.Open() && !svc.allows(r) {
			w.Header().Set("WWW-Authenticate", svc.challenge.Header())
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		jpeg, err := still()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpeg)
	}
}
