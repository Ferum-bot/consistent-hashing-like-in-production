package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/commands"
)

func main() {
	routers := urlList(defaultRouterURLs)
	wallets := urlList(defaultWalletURLs)
	loadgen := defaultLoadgenURL

	flag.Var(&routers, routersFlag, "comma separated router `urls`")
	flag.Var(&wallets, walletsFlag, "comma separated wallet `urls`, used by status")
	flag.StringVar(&loadgen, loadgenFlag, loadgen, "loadgen `url`")

	flag.Usage = func() {
		commands.PrintUsage(os.Stderr)
		flag.PrintDefaults()
	}

	flag.Parse()

	command, args, err := commands.Find(flag.Args())
	if err != nil {
		exitWithUsage(err)
	}

	env := newEnv(routers, wallets, loadgen)

	if len(env.Routers) == 0 {
		exitWithUsage(commands.UsageError{Reason: "-" + routersFlag + " needs at least one url"})
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err = command.Run(ctx, env, args)

	var usage commands.UsageError

	switch {
	case err == nil:
	case errors.Is(err, flag.ErrHelp):
	case errors.As(err, &usage):
		exitWithCommandUsage(command, usage)
	default:
		fmt.Fprintln(os.Stderr, "chctl: "+commands.Explain(err))
		os.Exit(1)
	}
}

func newEnv(routers, wallets urlList, loadgen string) commands.Env {
	clients := client.NewFactory(&http.Client{})

	env := commands.Env{
		Clients: clients,
		Loadgen: clients.Loadgen(loadgen),
		Out:     os.Stdout,
	}

	for _, url := range routers {
		env.Routers = append(env.Routers, clients.Router(url))
	}

	for _, url := range wallets {
		env.Wallets = append(env.Wallets, clients.Wallet(url))
	}

	return env
}

func exitWithUsage(err error) {
	var usage commands.UsageError
	if errors.As(err, &usage) && usage.Reason != "" {
		fmt.Fprintf(os.Stderr, "chctl: %s\n\n", usage.Reason)
	}

	flag.Usage()
	os.Exit(2)
}

func exitWithCommandUsage(command commands.Command, usage commands.UsageError) {
	if usage.Reason != "" {
		fmt.Fprintf(os.Stderr, "chctl: %s\n", usage.Reason)
	}

	fmt.Fprintf(os.Stderr, "usage: chctl %s\n", command.Usage())
	os.Exit(2)
}
