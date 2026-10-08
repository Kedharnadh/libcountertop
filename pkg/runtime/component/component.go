package component

import (
	"context"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/runtime/service"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
)

type Phase int

const (
	Hardware Phase = iota
	Device
	Network
)

func (p Phase) String() string {
	switch p {
	case Hardware:
		return "hardware"
	case Device:
		return "device"
	case Network:
		return "network"
	}
	return "unknown"
}

type Component interface{ Name() string }
type Starter = service.Starter
type Closer = service.Closer
type Runner interface{ Run(context.Context) error }
type Entities interface{ Entities() []esphome.Entity }
type Actions interface{ Actions() []*esphome.Action }
type Handler interface{ esphome.Handler }
type Progress = startup.Progress
type Startup interface{ Startup() Progress }
type Restorer[T any] interface{ Restore(T) }
type Event struct {
	Name string
	Data map[string]string
}
