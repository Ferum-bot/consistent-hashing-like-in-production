package repositories

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

const twoUsers = `{"user_id":1,"wallets":[{"currency":"USD","balance":100},{"currency":"EUR","balance":50}]}
{"user_id":2,"wallets":[{"currency":"JPY","balance":7}]}
`

func loadLedger(t *testing.T, body string) *LedgerRepository {
	t.Helper()

	path := filepath.Join(t.TempDir(), "snapshot.ndjson")

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	ledger := NewLedgerRepository()

	if err := ledger.Load(context.Background(), path); err != nil {
		t.Fatal(err)
	}

	return ledger
}

func TestLedger_LoadAndApply(t *testing.T) {
	ledger := loadLedger(t, twoUsers)

	stats := ledger.Stats()
	if stats != (models.LedgerStats{Loaded: true, Users: 2, Wallets: 3, Money: 157}) {
		t.Fatalf("stats = %+v", stats)
	}

	ticket := ledger.BeginWrite(1)
	ledger.Apply(ticket, "EUR", -20)
	ledger.EndWrite(ticket)

	wallets := ledger.Wallets(1)
	if wallets[1] != (models.Wallet{Currency: "EUR", Balance: 30}) {
		t.Fatalf("wallets = %+v", wallets)
	}

	if ledger.Wallets(3) != nil {
		t.Fatal("unknown user must have no wallets")
	}
}

func TestLedger_ObservationSeesWrites(t *testing.T) {
	ledger := loadLedger(t, twoUsers)

	before := ledger.Observe(1)
	if before.Busy || !ledger.Unchanged(before) {
		t.Fatalf("fresh observation must be usable: %+v", before)
	}

	ticket := ledger.BeginWrite(1)

	if !ledger.Observe(1).Busy {
		t.Fatal("user with a write in flight must be busy")
	}

	ledger.EndWrite(ticket)

	if ledger.Unchanged(before) {
		t.Fatal("a write between observations must be noticed")
	}

	uncertain := ledger.BeginWrite(2)
	ledger.MarkUncertain(uncertain)
	ledger.EndWrite(uncertain)

	if !ledger.Observe(2).Busy || ledger.UncertainUsers() != 1 {
		t.Fatal("uncertain user must stay busy")
	}
}

func TestLedger_ReloadIgnoresOldTickets(t *testing.T) {
	ledger := loadLedger(t, twoUsers)
	ticket := ledger.BeginWrite(1)

	path := filepath.Join(t.TempDir(), "again.ndjson")
	if err := os.WriteFile(path, []byte(twoUsers), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ledger.Load(context.Background(), path); err != nil {
		t.Fatal(err)
	}

	ledger.Apply(ticket, "USD", 1000)
	ledger.EndWrite(ticket)

	if ledger.Wallets(1)[0].Balance != 100 || ledger.Observe(1).Busy {
		t.Fatal("a ticket from the previous load must not touch the new ledger")
	}
}

func TestLedger_RejectsUnsortedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.ndjson")
	body := `{"user_id":2,"wallets":[]}` + "\n"

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	err := NewLedgerRepository().Load(context.Background(), path)
	if !errors.Is(err, models.ErrSnapshotNotSorted) {
		t.Fatalf("err = %v, want ErrSnapshotNotSorted", err)
	}
}
