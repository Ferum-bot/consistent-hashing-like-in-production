package commands

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

var abortCommand = Command{
	Name:    "abort",
	Summary: "roll back an unfinished migration on every router",
	Run:     runAbort,
}

func runAbort(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	return abortEverywhere(ctx, env)
}

func abortEverywhere(ctx context.Context, env Env) error {
	failed := 0

	for _, router := range env.Routers {
		change, err := router.Abort(ctx)

		switch {
		case client.HasStatus(err, http.StatusConflict):
			fmt.Fprintf(env.Out, "  %s  nothing to abort\n", router.URL())
		case err != nil:
			fmt.Fprintf(env.Out, "  %s  %s\n", router.URL(), err)
			failed++
		case change.Deferred():
			fmt.Fprintf(env.Out, "  %s  %s\n", router.URL(), describeChange(change))
		default:
			fmt.Fprintf(env.Out, "  %s  aborted, back on %s\n", router.URL(), describeChange(change))
		}
	}

	if failed > 0 {
		return fmt.Errorf("abort failed on %d of %d routers", failed, len(env.Routers))
	}

	return nil
}
