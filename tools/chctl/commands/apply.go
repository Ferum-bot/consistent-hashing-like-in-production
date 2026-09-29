package commands

import (
	"context"
	"fmt"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

var applyCommand = Command{
	Name:    "apply",
	Args:    "<topology.json>",
	Summary: "apply a topology to every router; with \"migrate\": true the first router copies the data",
	Run:     runApply,
}

func runApply(ctx context.Context, env Env, args []string) error {
	if len(args) != 1 {
		return UsageError{}
	}

	file, err := readTopologyFile(args[0])
	if err != nil {
		return err
	}

	if err := waitReady(ctx, env.Out, env.Routers...); err != nil {
		return err
	}

	states, err := readRouters(ctx, env.Routers)
	if err != nil {
		return err
	}

	if err := unfinishedMigration(states); err != nil {
		return err
	}

	if err := catchUpRestarted(ctx, env, states); err != nil {
		return err
	}

	topology := file.Topology
	topology.Version = nextVersion(states)

	if !file.Migrate {
		return applyEverywhere(ctx, env, topology)
	}

	if !anyTopology(states) {
		fmt.Fprintln(env.Out, "routers have no topology yet, nothing to migrate: applying directly")

		return applyEverywhere(ctx, env, topology)
	}

	return migrate(ctx, env, topology)
}

func catchUpRestarted(ctx context.Context, env Env, states []routerState) error {
	source, ok := newestState(states)
	if !ok {
		return nil
	}

	current := source.topology
	current.PendingVersion = 0
	current.DeferredError = ""
	current.MembershipDelay = nil

	for i, state := range states {
		if state.set {
			continue
		}

		fmt.Fprintf(env.Out, "%s has no topology, probably restarted: giving it v%d from %s\n", state.router.URL(), current.Version, source.router.URL())

		if _, err := state.router.Apply(ctx, current); err != nil {
			return err
		}

		states[i].topology = current
		states[i].set = true
	}

	return nil
}

func applyEverywhere(ctx context.Context, env Env, topology client.Topology) error {
	fmt.Fprintf(env.Out, "applying v%d: %s\n", topology.Version, describeTopology(topology))

	for _, router := range env.Routers {
		change, err := router.Apply(ctx, topology)
		if err != nil {
			return err
		}

		fmt.Fprintf(env.Out, "  %s  %s\n", router.URL(), describeChange(change))
	}

	return nil
}
