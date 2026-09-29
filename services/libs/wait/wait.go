package wait

import (
	"context"
	"time"
)

func For(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func Until(ctx context.Context, at time.Time) bool {
	return For(ctx, time.Until(at))
}
