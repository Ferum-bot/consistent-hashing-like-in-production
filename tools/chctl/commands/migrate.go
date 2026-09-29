package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

const (
	switchWait = 10 * time.Minute
	switchPoll = time.Second
)

func migrate(ctx context.Context, env Env, topology client.Topology) error {
	coordinator := env.Routers[0]
	start := time.Now()

	fmt.Fprintf(env.Out, "migrating to v%d: %s\n\n", topology.Version, describeTopology(topology))

	err := phase(env, "prepare: every router learns the new topology, writes to moving users wait", func() error {
		return everyRouter(env, func(router *client.Router) (client.Change, error) {
			return router.Prepare(ctx, topology)
		})
	})
	if err != nil {
		return rollBack(ctx, env, err)
	}

	err = phase(env, "copy: moving users go to their new nodes", func() error {
		copied, err := coordinator.Migrate(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintf(env.Out, "  copied %d users%s\n", copied.Users, describeSpread(copied))

		return nil
	})
	if err != nil {
		return rollBack(ctx, env, err)
	}

	err = phase(env, "commit: every router switches, waiting writes go on", func() error {
		return everyRouter(env, func(router *client.Router) (client.Change, error) {
			return router.Commit(ctx, topology.Version)
		})
	})
	if err != nil {
		return withHint(err, "some routers may still wait on the old topology; "+hintStatus)
	}

	fmt.Fprintf(env.Out, "writes to moving users were held up to %s, from prepare to commit\n\n", time.Since(start).Round(time.Millisecond))

	return finishMigration(ctx, env, topology.Version)
}

func finishMigration(ctx context.Context, env Env, version int64) error {
	err := phase(env, "wait: every router is on the new topology, nobody writes to old copies", func() error {
		return waitSwitched(ctx, env, version)
	})
	if err != nil && hintFor(err) == "" {
		return withHint(err, "old copies stay on the old nodes until every router switches, then run: chctl cleanup")
	}

	if err != nil {
		return err
	}

	return phase(env, "cleanup: moved users are removed from old nodes", func() error {
		removed, err := env.Routers[0].Cleanup(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintf(env.Out, "  removed %d old copies\n", removed)

		return nil
	})
}

func phase(env Env, name string, run func() error) error {
	fmt.Fprintf(env.Out, "== %s\n", name)

	start := time.Now()

	if err := run(); err != nil {
		return err
	}

	fmt.Fprintf(env.Out, "== done in %s\n\n", time.Since(start).Round(time.Millisecond))

	return nil
}

func everyRouter(env Env, change func(router *client.Router) (client.Change, error)) error {
	for _, router := range env.Routers {
		result, err := change(router)
		if err != nil {
			return err
		}

		fmt.Fprintf(env.Out, "  %s  %s\n", router.URL(), describeChange(result))
	}

	return nil
}

func rollBack(ctx context.Context, env Env, cause error) error {
	fmt.Fprintln(env.Out, "== abort: routers go back to the current topology")

	if err := abortEverywhere(context.WithoutCancel(ctx), env); err != nil {
		return errors.Join(cause, fmt.Errorf("roll back: %w", err))
	}

	return cause
}

func waitSwitched(ctx context.Context, env Env, version int64) error {
	deadline := time.Now().Add(switchWait)

	for _, router := range env.Routers {
		reported := false

		for {
			topology, _, err := router.Topology(ctx)
			if err != nil {
				return err
			}

			if topology.Version >= version {
				fmt.Fprintf(env.Out, "  %s is on v%d\n", router.URL(), topology.Version)
				break
			}

			if topology.DeferredError != "" {
				err := fmt.Errorf("%s could not apply its delayed change: %s", router.URL(), topology.DeferredError)

				return withHint(err, "the router stays on v"+strconv.FormatInt(topology.Version, 10)+"; fix the cause and apply the topology again")
			}

			if !reported {
				fmt.Fprintf(env.Out, "  %s is still on v%d, waiting\n", router.URL(), topology.Version)
				reported = true
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("%s did not switch to v%d in %s", router.URL(), version, switchWait)
			}

			if !sleep(ctx, switchPoll) {
				return ctx.Err()
			}
		}
	}

	return nil
}
