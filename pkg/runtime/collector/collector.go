package collector

import (
	"context"
	"time"
)

func Run(
	ctx context.Context,
	interval func() time.Duration,
	wake <-chan struct{},
	sample func(),
) error {
	for ctx.Err() == nil {
		sample()
		timer := time.NewTimer(max(interval(), time.Second))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-wake:
			timer.Stop()
		case <-timer.C:
		}
	}
	return nil
}
