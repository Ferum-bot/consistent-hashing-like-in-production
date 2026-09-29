package commands

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

var bootstrapCommand = Command{
	Name:    "bootstrap",
	Summary: "spread the snapshot over wallet nodes by the current topology, through the first router",
	Run:     runBootstrap,
}

func runBootstrap(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	router := env.Routers[0]
	start := time.Now()

	fmt.Fprintf(env.Out, "spreading the snapshot through %s\n", router.URL())

	spread, err := router.Bootstrap(ctx)

	switch {
	case client.HasStatus(err, http.StatusServiceUnavailable):
		return withHint(err, hintApplyFirst)
	case client.HasStatus(err, http.StatusInternalServerError):
		return withHint(err, "the router reads data/snapshot.ndjson; "+hintGenerateData)
	case client.HasStatus(err, http.StatusBadGateway):
		return withHint(err, "a wallet node from the topology is down; "+hintStatus)
	case err != nil:
		return err
	}

	fmt.Fprintf(env.Out, "spread %d users in %s%s\n", spread.Users, time.Since(start).Round(time.Millisecond), describeSpread(spread))

	return nil
}
