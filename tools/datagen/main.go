package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
)

var currencies = []string{"USD", "EUR", "GBP", "JPY", "CHF", "CNY", "AED", "TRY"}

type wallet struct {
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
}

type user struct {
	UserID  int64    `json:"user_id"`
	Wallets []wallet `json:"wallets"`
}

func main() {
	users := flag.Int64("users", 2_000_000, "number of users")
	maxWallets := flag.Int("max-wallets", 5, "max wallets per user, at least one")
	maxBalance := flag.Int64("max-balance", 1_000_000, "max balance of a wallet in minor units")
	seed := flag.Uint64("seed", 1, "random seed")
	out := flag.String("out", "../data/snapshot.ndjson", "output file")
	flag.Parse()

	if *users < 1 || *maxWallets < 1 || *maxBalance < 0 {
		fmt.Fprintln(os.Stderr, "datagen: -users and -max-wallets must be at least 1, -max-balance must not be negative")
		os.Exit(2)
	}

	if err := generate(*users, *maxWallets, *maxBalance, *seed, *out); err != nil {
		fmt.Fprintln(os.Stderr, "datagen:", err)
		os.Exit(1)
	}
}

func generate(users int64, maxWallets int, maxBalance int64, seed uint64, out string) error {
	maxWallets = min(maxWallets, len(currencies))

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}

	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1<<20)
	encoder := json.NewEncoder(writer)
	random := rand.New(rand.NewPCG(seed, seed))
	wallets := 0

	for id := int64(1); id <= users; id++ {
		next := randomUser(random, id, maxWallets, maxBalance)

		if err := encoder.Encode(next); err != nil {
			return err
		}

		wallets += len(next.Wallets)
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	fmt.Printf("%d users, %d wallets written to %s\n", users, wallets, out)

	return nil
}

func randomUser(random *rand.Rand, id int64, maxWallets int, maxBalance int64) user {
	count := 1 + random.IntN(maxWallets)
	order := random.Perm(len(currencies))
	wallets := make([]wallet, count)

	for i := range wallets {
		wallets[i] = wallet{
			Currency: currencies[order[i]],
			Balance:  random.Int64N(maxBalance + 1),
		}
	}

	return user{UserID: id, Wallets: wallets}
}
