package detect

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/zserge/microwakeword"

	"github.com/ygelfand/libcountertop/pkg/inference/oww"
	"github.com/ygelfand/libcountertop/pkg/inference/wake"
)

type backend interface {
	load(m wake.Model) error

	unload(id string)

	feed(frame []int16) (map[string]float64, bool)

	close()
}

func newBackend(k wake.Kind) (backend, error) {
	if k == wake.KindOpenWakeWord {
		front, err := oww.New()
		if err != nil {
			return nil, err
		}
		return &owwBackend{front: front, scores: map[string]float64{}}, nil
	}
	return &microBackend{
		dets:   map[string]*microwakeword.Detector{},
		scores: map[string]float64{},
	}, nil
}

type microBackend struct {
	dets   map[string]*microwakeword.Detector
	scores map[string]float64
}

const neverFires = 1.1

func (b *microBackend) load(m wake.Model) error {
	cfg := m.Config
	cfg.ProbabilityCutoff = neverFires

	det, err := microwakeword.NewDetector(cfg)
	if err != nil {
		return fmt.Errorf("wake: loading %s: %w", m.Path, err)
	}
	b.dets[m.ID] = det
	return nil
}

func (b *microBackend) unload(id string) {
	delete(b.dets, id)
	delete(b.scores, id)
}

func (b *microBackend) feed(frame []int16) (map[string]float64, bool) {
	for id, det := range b.dets {
		det.ProcessAudio(frame)
		b.scores[id] = det.SlidingAverage()
	}
	return b.scores, true
}

func (b *microBackend) close() { clear(b.dets) }

type owwBackend struct {
	front  *oww.Engine
	scores map[string]float64
}

func (b *owwBackend) load(m wake.Model) error {
	model, err := os.ReadFile(m.Path)
	if err != nil {
		return fmt.Errorf("wake: reading %s: %w", m.Path, err)
	}
	_, err = b.front.Load(m.ID, model)
	return err
}

func (b *owwBackend) unload(id string) {
	b.front.Unload(id)
	delete(b.scores, id)
}

func (b *owwBackend) feed(frame []int16) (map[string]float64, bool) {
	scores, err := b.front.Process(frame)
	if err != nil {
		slog.Error("wake scoring failed", "err", err)
		return nil, false
	}
	if len(scores) == 0 {
		return nil, false
	}

	for id, v := range scores {
		b.scores[id] = float64(v)
	}
	return b.scores, true
}

func (b *owwBackend) close() {}
