package commands

import (
	"context"
)

var cleanupCommand = Command{
	Name:    "cleanup",
	Summary: "finish an interrupted migration: wait for every router to switch, then remove old copies through the first router",
	Run:     runCleanup,
}

func runCleanup(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	states, err := readRouters(ctx, env.Routers)
	if err != nil {
		return err
	}

	if err := unfinishedMigration(states); err != nil {
		return err
	}

	return finishMigration(ctx, env, newestVersion(states))
}
