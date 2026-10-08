package firmware

import (
	"context"
	"time"
)

// ESPHome update entities are not polled.
const (
	checkEvery  = 24 * time.Hour
	checkSettle = 5 * time.Minute
)

func (u *Firmware[M]) Run(ctx context.Context) error {
	first := time.NewTimer(checkSettle)
	defer first.Stop()

	select {
	case <-ctx.Done():
		return nil
	case <-first.C:
		if t, ok := u.deps.Source.(Trial); ok {
			t.Commit()
		}
		u.Check(ctx)
	}

	t := time.NewTicker(checkEvery)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			u.Check(ctx)
		}
	}
}
