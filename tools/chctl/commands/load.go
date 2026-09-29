package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

var loadStatusCommand = Command{
	Name:    "load status",
	Summary: "show the load profile and the ledger of expected balances",
	Run:     runLoadStatus,
}

var loadSetCommand = Command{
	Name:    "load set",
	Args:    "key=value ...",
	Summary: "change the load profile, e.g. running=true rps=2000",
	Run:     runLoadSet,
}

var loadApplyCommand = Command{
	Name:    "load apply",
	Args:    "<load.json>",
	Summary: "change the load profile from a file",
	Run:     runLoadApply,
}

var loadPauseCommand = Command{
	Name:    "load pause",
	Summary: "pause the load, does nothing when loadgen is not running",
	Run:     runLoadPause,
}

var loadReloadCommand = Command{
	Name:    "load reload",
	Summary: "reload the ledger from the snapshot, needed after reset and bootstrap",
	Run:     runLoadReload,
}

func runLoadStatus(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	profile, err := env.Loadgen.Profile(ctx)
	if err != nil {
		return err
	}

	ledger, loaded, err := env.Loadgen.Ledger(ctx)
	if err != nil {
		return err
	}

	printProfile(env, profile)
	fmt.Fprintln(env.Out, describeLedger(ledger, loaded))

	if !loaded {
		fmt.Fprintln(env.Out, "hint: "+hintReloadLedger)
	}

	return nil
}

func runLoadSet(ctx context.Context, env Env, args []string) error {
	if len(args) == 0 {
		return UsageError{}
	}

	patch := map[string]any{}

	for _, pair := range args {
		key, raw, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return wrongUsage("%q must look like key=value", pair)
		}

		var value any
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			value = raw
		}

		patch[key] = value
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	return updateProfile(ctx, env, body)
}

func runLoadApply(ctx context.Context, env Env, args []string) error {
	if len(args) != 1 {
		return UsageError{}
	}

	body, err := os.ReadFile(args[0])
	if errors.Is(err, fs.ErrNotExist) {
		return withHint(err, "scenarios keep it in scenarios/<name>/load.json")
	}

	if err != nil {
		return err
	}

	if !json.Valid(body) {
		return fmt.Errorf("%s is not valid json", args[0])
	}

	if err := waitReady(ctx, env.Out, env.Loadgen); err != nil {
		return err
	}

	return updateProfile(ctx, env, body)
}

func runLoadPause(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	if err := env.Loadgen.Ready(ctx); client.IsUnreachable(err) {
		fmt.Fprintln(env.Out, "loadgen is not running, nothing to pause")
		return nil
	}

	return updateProfile(ctx, env, json.RawMessage(`{"running": false}`))
}

func runLoadReload(ctx context.Context, env Env, args []string) error {
	if len(args) != 0 {
		return UsageError{}
	}

	fmt.Fprintln(env.Out, "reloading the ledger from the snapshot")

	ledger, err := env.Loadgen.ReloadLedger(ctx)
	if client.HasStatus(err, http.StatusInternalServerError) {
		return withHint(err, hintGenerateData)
	}

	if err != nil {
		return err
	}

	fmt.Fprintln(env.Out, describeLedger(ledger, true))

	return nil
}

func updateProfile(ctx context.Context, env Env, patch json.RawMessage) error {
	profile, err := env.Loadgen.UpdateProfile(ctx, patch)
	if err != nil {
		return err
	}

	printProfile(env, profile)

	return nil
}

func printProfile(env Env, profile client.Profile) {
	fmt.Fprintln(env.Out, describeProfile(profile))
	fmt.Fprintf(env.Out,
		"  read_ratio %g  deposit_share %g  read_zipf_s %g  write_zipf_s %g  hot_share %g  merchant %d  max_amount %d  verify_rps %d\n",
		profile.ReadRatio, profile.DepositShare, profile.ReadZipfS, profile.WriteZipfS,
		profile.HotShare, profile.MerchantUserID, profile.MaxAmount, profile.VerifyRPS,
	)
}

func describeProfile(profile client.Profile) string {
	if !profile.Running {
		return fmt.Sprintf("paused, %d rps when running", profile.RPS)
	}

	return fmt.Sprintf("running %d rps", profile.RPS)
}

func describeLedger(ledger client.Ledger, loaded bool) string {
	if !loaded {
		return "ledger not loaded"
	}

	return fmt.Sprintf("ledger %d users, %d wallets, %d uncertain", ledger.Users, ledger.Wallets, ledger.Uncertain)
}
