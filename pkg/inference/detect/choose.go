package detect

import (
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/inference/wake"
)

type Slots interface {
	Use(int, wake.Model) error
	Clear(int)
}

func Choose(
	engine Slots,
	models []wake.Model,
	ids []string,
	count int,
) (accepted []string, loaded []int) {
	accepted = make([]string, count)
	for slot := range count {
		if slot >= len(ids) || ids[slot] == "" {
			engine.Clear(slot)
			continue
		}

		m, ok := wake.Find(models, ids[slot])
		if !ok {
			slog.Warn("a chosen wake word is not on the device", "slot", slot+1, "id", ids[slot])
			engine.Clear(slot)
			continue
		}

		if err := engine.Use(slot, m); err != nil {
			slog.Error("loading a wake word failed", "slot", slot+1, "id", m.ID, "err", err)
			engine.Clear(slot)
			continue
		}
		accepted[slot] = m.ID
		loaded = append(loaded, slot)
	}
	return accepted, loaded
}

func LogInstalled(ours []wake.Model) {
	slog.Info("wake words installed", "count", len(ours),
		"openwakeword", len(wake.OfKind(ours, wake.KindOpenWakeWord)),
		"microwakeword", len(wake.OfKind(ours, wake.KindMicroWakeWord)))
}
