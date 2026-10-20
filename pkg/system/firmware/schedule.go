package firmware

import (
	"context"
	"time"
)

// ESPHome update entities are not polled.
const (
	checkEvery  = 24 * time.Hour
	checkSettle = 5 * time.Minute
	listRetry   = 30 * time.Second
)

func (u *Firmware[M]) Run(ctx context.Context) error {
	list := time.NewTimer(0)
	defer list.Stop()
	first := time.NewTimer(checkSettle)
	defer first.Stop()
	daily := time.NewTicker(checkEvery)
	defer daily.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-list.C:
			if u.List(ctx) != nil {
				list.Reset(listRetry)
			}
		case <-first.C:
			if t, ok := u.deps.Source.(Trial); ok {
				t.Commit()
			}
			u.Check(ctx)
		case <-daily.C:
			u.Check(ctx)
		}
	}
}
