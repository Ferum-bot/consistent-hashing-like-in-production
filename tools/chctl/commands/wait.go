package commands

import (
	"context"
	"fmt"
	"io"
	"time"
)

const (
	startupWait = time.Minute
	startupPoll = 500 * time.Millisecond
)

type service interface {
	Service() string
	URL() string
	Ready(ctx context.Context) error
}

func waitReady[S service](ctx context.Context, out io.Writer, services ...S) error {
	deadline := time.Now().Add(startupWait)

	for _, s := range services {
		err := s.Ready(ctx)
		if err == nil {
			continue
		}

		fmt.Fprintf(out, "waiting up to %s for %s %s to start\n", startupWait, s.Service(), s.URL())

		for err != nil {
			if time.Now().After(deadline) {
				return err
			}

			if !sleep(ctx, startupPoll) {
				return ctx.Err()
			}

			err = s.Ready(ctx)
		}
	}

	return nil
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
