package commands

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

const (
	defaultWalletURL    = "http://localhost:8080"
	defaultSnapshotPath = "../data/snapshot.ndjson"
	defaultImportBatch  = 5000
	maxSnapshotLine     = 1 << 20
	progressEvery       = 100_000
	importArgs          = "[-snapshot file] [-batch n] [-limit n] [wallet url]"
)

var importCommand = Command{
	Name:    "import",
	Args:    importArgs,
	Summary: "load the snapshot straight into one wallet, without routers",
	Run:     runImport,
}

type importOptions struct {
	snapshot string
	batch    int
	limit    int
	url      string
}

func runImport(ctx context.Context, env Env, args []string) error {
	options, err := parseImportOptions(env.Out, args)
	if err != nil {
		return err
	}

	file, err := os.Open(options.snapshot)
	if errors.Is(err, fs.ErrNotExist) {
		return withHint(err, hintGenerateData)
	}

	if err != nil {
		return err
	}
	defer file.Close()

	wallet := env.Clients.Wallet(options.url)

	if err := wallet.Ready(ctx); err != nil {
		return err
	}

	start := time.Now()

	imported, err := importSnapshot(ctx, env, wallet, file, options)
	if client.HasStatus(err, http.StatusConflict) {
		return withHint(err, "the wallet is in strict fencing mode and accepts only routers, restart it to start clean")
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(env.Out, "imported %d users into %s in %s\n", imported, wallet.URL(), time.Since(start).Round(time.Millisecond))

	return nil
}

func parseImportOptions(out io.Writer, args []string) (importOptions, error) {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	options := importOptions{url: defaultWalletURL}

	flags.StringVar(&options.snapshot, "snapshot", defaultSnapshotPath, "snapshot file")
	flags.IntVar(&options.batch, "batch", defaultImportBatch, "users per request")
	flags.IntVar(&options.limit, "limit", 0, "import only the first N users, 0 means all")

	err := flags.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		printImportHelp(out, flags)
		return importOptions{}, err
	}

	if err != nil {
		return importOptions{}, UsageError{Reason: err.Error()}
	}

	if flags.NArg() > 1 || options.batch < 1 || options.limit < 0 {
		return importOptions{}, UsageError{}
	}

	if flags.NArg() == 1 {
		options.url = flags.Arg(0)
	}

	return options, nil
}

func printImportHelp(out io.Writer, flags *flag.FlagSet) {
	fmt.Fprintf(out, "usage: chctl import %s\n\n", importArgs)
	fmt.Fprintf(out, "wallet url defaults to %s\n\n", defaultWalletURL)

	flags.SetOutput(out)
	flags.PrintDefaults()
}

func importSnapshot(ctx context.Context, env Env, wallet *client.Wallet, file *os.File, options importOptions) (int, error) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSnapshotLine)

	var (
		batch    bytes.Buffer
		pending  int
		imported int
	)

	flush := func() error {
		if pending == 0 {
			return nil
		}

		if _, err := wallet.Import(ctx, batch.Bytes()); err != nil {
			return err
		}

		imported += pending
		pending = 0
		batch.Reset()

		if imported%progressEvery < options.batch {
			fmt.Fprintf(env.Out, "imported %d users\n", imported)
		}

		return nil
	}

	for scanner.Scan() {
		if options.limit > 0 && imported+pending >= options.limit {
			break
		}

		batch.Write(scanner.Bytes())
		batch.WriteByte('\n')
		pending++

		if pending < options.batch {
			continue
		}

		if err := flush(); err != nil {
			return imported, err
		}
	}

	if err := scanner.Err(); err != nil {
		return imported, err
	}

	return imported, flush()
}
