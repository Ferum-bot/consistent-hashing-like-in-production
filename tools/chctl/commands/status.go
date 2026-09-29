package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

var statusCommand = Command{
	Name:    "status",
	Summary: "show what is running: routers and their topology, wallets, loadgen",
	Run:     runStatus,
}

type statusReport struct {
	table    *tabwriter.Writer
	running  int
	versions map[int64]bool
}

func runStatus(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	report := &statusReport{
		table:    tabwriter.NewWriter(env.Out, 0, 4, 2, ' ', 0),
		versions: map[int64]bool{},
	}

	report.section("ROUTERS")

	for _, router := range env.Routers {
		report.router(ctx, router)
	}

	report.section("WALLETS")

	for _, wallet := range env.Wallets {
		report.wallet(ctx, wallet)
	}

	report.section("LOADGEN")
	report.loadgen(ctx, env.Loadgen)

	if err := report.table.Flush(); err != nil {
		return err
	}

	report.notes(env.Out)

	return nil
}

func (r *statusReport) section(title string) {
	fmt.Fprintln(r.table, title)
}

func (r *statusReport) row(url, state, details string) {
	fmt.Fprintf(r.table, "  %s\t%s\t%s\n", url, state, details)
}

func (r *statusReport) down(url string, err error) {
	var unreachable *client.UnreachableError
	if errors.As(err, &unreachable) {
		r.row(url, "down", unreachable.Reason())
		return
	}

	r.row(url, "error", err.Error())
}

func (r *statusReport) router(ctx context.Context, router *client.Router) {
	topology, set, err := router.Topology(ctx)
	if err != nil {
		r.down(router.URL(), err)
		return
	}

	r.running++

	if !set {
		r.row(router.URL(), "up", "no topology yet")
		return
	}

	r.versions[topology.Version] = true

	version := fmt.Sprintf("v%d", topology.Version)

	if topology.PendingVersion != 0 {
		version += fmt.Sprintf(", migrating to v%d", topology.PendingVersion)
	}

	details := version + "  " + describeTopology(topology)

	if topology.DeferredError != "" {
		details += "  delayed change failed: " + topology.DeferredError
	}

	r.row(router.URL(), "up", details)
}

func (r *statusReport) wallet(ctx context.Context, wallet *client.Wallet) {
	stats, err := wallet.Stats(ctx)
	if err != nil {
		r.down(wallet.URL(), err)
		return
	}

	r.running++

	details := fmt.Sprintf("%s  %d users  %d wallets", stats.Node, stats.Users, stats.Wallets)

	switch {
	case stats.Strict:
		details += fmt.Sprintf("  strict, v%d", stats.TopologyVersion)
	case stats.TopologyVersion != 0:
		details += fmt.Sprintf("  saw v%d", stats.TopologyVersion)
	}

	r.row(wallet.URL(), "up", details)
}

func (r *statusReport) loadgen(ctx context.Context, loadgen *client.Loadgen) {
	profile, err := loadgen.Profile(ctx)
	if err != nil {
		r.down(loadgen.URL(), err)
		return
	}

	r.running++

	ledger, loaded, err := loadgen.Ledger(ctx)
	if err != nil {
		r.down(loadgen.URL(), err)
		return
	}

	details := describeProfile(profile) + "  " + describeLedger(ledger, loaded)

	r.row(loadgen.URL(), "up", details)
}

func (r *statusReport) notes(out io.Writer) {
	if r.running == 0 {
		fmt.Fprintln(out, "\nnothing is running\nhint: "+hintStartStand)
		return
	}

	if len(r.versions) > 1 {
		fmt.Fprintln(out, "\nrouters are on different topology versions, a lagging router catches up on its own")
	}
}
