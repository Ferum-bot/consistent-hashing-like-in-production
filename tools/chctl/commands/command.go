package commands

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

type UsageError struct {
	Reason string
}

func (e UsageError) Error() string {
	if e.Reason == "" {
		return "wrong usage"
	}

	return e.Reason
}

func wrongUsage(format string, args ...any) error {
	return UsageError{Reason: fmt.Sprintf(format, args...)}
}

type Env struct {
	Clients client.Factory
	Routers []*client.Router
	Wallets []*client.Wallet
	Loadgen *client.Loadgen
	Out     io.Writer
}

type Command struct {
	Name    string
	Args    string
	Summary string
	Run     func(ctx context.Context, env Env, args []string) error
}

func List() []Command {
	return []Command{
		statusCommand,
		applyCommand,
		abortCommand,
		cleanupCommand,
		bootstrapCommand,
		importCommand,
		loadStatusCommand,
		loadSetCommand,
		loadApplyCommand,
		loadPauseCommand,
		loadReloadCommand,
	}
}

func Find(args []string) (Command, []string, error) {
	if len(args) == 0 {
		return Command{}, nil, UsageError{}
	}

	for _, command := range List() {
		words := strings.Fields(command.Name)

		if len(args) >= len(words) && slices.Equal(args[:len(words)], words) {
			return command, args[len(words):], nil
		}
	}

	for _, command := range List() {
		if !strings.HasPrefix(command.Name, args[0]+" ") {
			continue
		}

		if len(args) == 1 {
			return Command{}, nil, wrongUsage("%q needs a subcommand", args[0])
		}

		return Command{}, nil, wrongUsage("unknown subcommand %q of %q", args[1], args[0])
	}

	return Command{}, nil, wrongUsage("unknown command %q", args[0])
}

func (c Command) Usage() string {
	return strings.TrimSpace(c.Name + " " + c.Args)
}

func PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: chctl [flags] <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")

	table := tabwriter.NewWriter(w, 0, 4, 3, ' ', 0)

	for _, command := range List() {
		fmt.Fprintf(table, "  %s\t%s\n", command.Usage(), command.Summary)
	}

	table.Flush()

	fmt.Fprintln(w)
	fmt.Fprintln(w, "flags:")
}
