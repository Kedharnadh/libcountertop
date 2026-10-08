package cast

import (
	"net/http"
	"sort"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/display/surface"
	"github.com/ygelfand/libcountertop/pkg/media/playback"
)

type Env struct {
	Name   string
	Model  string
	Device Device

	HTTP *http.Client
	Long *http.Client

	Video func() playback.Target

	Volume        func() int
	SetVolume     func(level int)
	VolumeChanged func(do func()) (stop func())

	Publish func(control Playing, p *Published)
	Send    func(m Message)
	Keep    func(app string) Kept

	Output playback.Output

	Resample func(from int) func(stereo []int16) []int16

	Surface func() *surface.Client
}

type Kept interface {
	Load(v any) bool
	Save(v any) error
	Clear() error
}

var (
	definedMu sync.Mutex
	defined   = map[string]func(Env) Protocol{}
)

func Define(name string, make func(Env) Protocol) {
	definedMu.Lock()
	defer definedMu.Unlock()
	defined[name] = make
}

func Protocols(env Env) []Protocol {
	definedMu.Lock()
	defer definedMu.Unlock()

	names := make([]string, 0, len(defined))
	for n := range defined {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]Protocol, 0, len(names))
	for _, n := range names {
		out = append(out, defined[n](env))
	}
	return out
}
