package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/repositories"
)

type fakeCluster struct {
	mu    sync.Mutex
	users map[int64]map[string]int64
}

func newFakeCluster(users int) (*fakeCluster, string) {
	cluster := &fakeCluster{users: map[int64]map[string]int64{}}

	var snapshot strings.Builder

	for id := int64(1); id <= int64(users); id++ {
		cluster.users[id] = map[string]int64{"USD": 1000, "EUR": 500}
		fmt.Fprintf(&snapshot, `{"user_id":%d,"wallets":[{"currency":"USD","balance":1000},{"currency":"EUR","balance":500}]}`+"\n", id)
	}

	return cluster, snapshot.String()
}

func (c *fakeCluster) Get(_ context.Context, _ string, userID int64) (models.UserResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	wallets, ok := c.users[userID]
	if !ok {
		return models.UserResponse{Status: http.StatusNotFound}, nil
	}

	response := models.UserResponse{Status: http.StatusOK}

	for currency, balance := range wallets {
		response.Wallets = append(response.Wallets, models.Wallet{Currency: currency, Balance: balance})
	}

	return response, nil
}

func (c *fakeCluster) Deposit(_ context.Context, _ string, userID int64, currency string, amount int64) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.users[userID][currency] += amount

	return http.StatusOK, nil
}

func (c *fakeCluster) Withdraw(_ context.Context, _ string, userID int64, currency string, amount int64) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.users[userID][currency] < amount {
		return http.StatusUnprocessableEntity, nil
	}

	c.users[userID][currency] -= amount

	return http.StatusOK, nil
}

type countingObserver struct {
	mu       sync.Mutex
	requests map[models.OperationKind]int
	checks   map[models.CheckResult]int
	dropped  int
}

func newCountingObserver() *countingObserver {
	return &countingObserver{requests: map[models.OperationKind]int{}, checks: map[models.CheckResult]int{}}
}

func (o *countingObserver) ObserveRequest(kind models.OperationKind, _ string, _ int, _ time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.requests[kind]++
}

func (o *countingObserver) ObserveDropped() {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.dropped++
}

func (o *countingObserver) ObserveCheck(result models.CheckResult) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.checks[result]++
}

func (o *countingObserver) ObserveProfile(models.Profile) {}

func defaultProfile() models.Profile {
	return models.Profile{
		Running:        true,
		RPS:            4000,
		ReadRatio:      0.5,
		DepositShare:   0.5,
		ReadZipfS:      1.2,
		WriteZipfS:     1.2,
		HotShare:       0.2,
		MerchantUserID: 3,
		MaxAmount:      300,
		VerifyRPS:      500,
	}
}

func loadedLedger(t *testing.T, snapshot string) *repositories.LedgerRepository {
	t.Helper()

	path := filepath.Join(t.TempDir(), "snapshot.ndjson")

	if err := os.WriteFile(path, []byte(snapshot), 0o600); err != nil {
		t.Fatal(err)
	}

	ledger := repositories.NewLedgerRepository()

	if err := ledger.Load(context.Background(), path); err != nil {
		t.Fatal(err)
	}

	return ledger
}

func TestGenerator_HonestClusterHasNoMismatches(t *testing.T) {
	cluster, snapshot := newFakeCluster(50)
	ledger := loadedLedger(t, snapshot)
	observer := newCountingObserver()

	generator, err := NewGenerator(
		ledger, cluster, observer, NewChecker(ledger, observer),
		[]string{"router-1", "router-2"}, 32, defaultProfile(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	generator.Run(ctx)

	observer.mu.Lock()
	defer observer.mu.Unlock()

	if observer.requests[models.OperationRead] == 0 || observer.requests[models.OperationDeposit] == 0 ||
		observer.requests[models.OperationWithdraw] == 0 || observer.requests[models.OperationVerify] == 0 {
		t.Fatalf("every kind of operation must be sent: %+v", observer.requests)
	}

	if observer.checks[models.CheckOK] == 0 {
		t.Fatalf("no successful checks: %+v", observer.checks)
	}

	for _, bad := range []models.CheckResult{models.CheckLost, models.CheckExtra, models.CheckMissing} {
		if observer.checks[bad] != 0 {
			t.Fatalf("honest cluster produced %d %s checks: %+v", observer.checks[bad], bad, observer.checks)
		}
	}
}

func TestChecker_FindsWhatWentWrong(t *testing.T) {
	_, snapshot := newFakeCluster(1)
	ledger := loadedLedger(t, snapshot)
	checker := NewChecker(ledger, newCountingObserver())

	wallets := func(usd, eur int64) []models.Wallet {
		return []models.Wallet{{Currency: "USD", Balance: usd}, {Currency: "EUR", Balance: eur}}
	}

	tests := []struct {
		name     string
		response models.UserResponse
		want     models.CheckResult
	}{
		{"same", models.UserResponse{Status: http.StatusOK, Wallets: wallets(1000, 500)}, models.CheckOK},
		{"less money", models.UserResponse{Status: http.StatusOK, Wallets: wallets(900, 500)}, models.CheckLost},
		{"more money", models.UserResponse{Status: http.StatusOK, Wallets: wallets(1000, 600)}, models.CheckExtra},
		{"wallet gone", models.UserResponse{Status: http.StatusOK, Wallets: wallets(1000, 500)[:1]}, models.CheckLost},
		{"user gone", models.UserResponse{Status: http.StatusNotFound}, models.CheckMissing},
		{"node down", models.UserResponse{Status: http.StatusBadGateway}, models.CheckSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checker.Check(ledger.Observe(1), tt.response); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}

	before := ledger.Observe(1)
	ticket := ledger.BeginWrite(1)
	ledger.EndWrite(ticket)

	if got := checker.Check(before, models.UserResponse{Status: http.StatusOK, Wallets: wallets(1, 1)}); got != models.CheckSkipped {
		t.Fatalf("a write during the read must skip the check, got %s", got)
	}
}

func TestPicker_FollowsProfile(t *testing.T) {
	_, snapshot := newFakeCluster(1000)
	ledger := loadedLedger(t, snapshot)

	profile := defaultProfile()
	profile.ReadRatio = 0.7
	profile.DepositShare = 0.5
	profile.HotShare = 0.1
	profile.MerchantUserID = 777

	pick := newPicker(rand.New(rand.NewPCG(1, 2)), profile, ledger.Users())

	const total = 100_000

	kinds := map[models.OperationKind]int{}
	merchant := 0
	writes := 0

	for range total {
		op := pick.next(ledger, time.Now())
		kinds[op.Kind]++

		if op.Kind == models.OperationRead {
			continue
		}

		writes++

		if op.UserID == profile.MerchantUserID {
			merchant++
		}

		if op.Currency != "USD" && op.Currency != "EUR" {
			t.Fatalf("currency %q is not one of the user's wallets", op.Currency)
		}

		if op.Amount < 1 || op.Amount > profile.MaxAmount {
			t.Fatalf("amount %d out of range", op.Amount)
		}
	}

	assertShare(t, "reads", kinds[models.OperationRead], total, 0.7)
	assertShare(t, "deposits among writes", kinds[models.OperationDeposit], writes, 0.5)
	assertShare(t, "merchant among writes", merchant, writes, 0.1)
}

func TestValidateProfile(t *testing.T) {
	tests := []func(*models.Profile){
		func(p *models.Profile) { p.RPS = -1 },
		func(p *models.Profile) { p.ReadRatio = 1.5 },
		func(p *models.Profile) { p.ReadZipfS = 0.5 },
		func(p *models.Profile) { p.WriteZipfS = 1 },
		func(p *models.Profile) { p.MaxAmount = 0 },
		func(p *models.Profile) { p.MerchantUserID = 0 },
	}

	for i, change := range tests {
		profile := defaultProfile()
		change(&profile)

		if err := validateProfile(profile); !errors.Is(err, models.ErrInvalidArgument) {
			t.Fatalf("case %d: err = %v, want ErrInvalidArgument", i, err)
		}
	}

	if err := validateProfile(defaultProfile()); err != nil {
		t.Fatal(err)
	}
}

func assertShare(t *testing.T, name string, got, total int, want float64) {
	t.Helper()

	share := float64(got) / float64(total)
	if share < want-0.01 || share > want+0.01 {
		t.Fatalf("%s: share %.3f, want about %.2f", name, share, want)
	}
}

type rejectingCluster struct {
	*fakeCluster
}

func (c rejectingCluster) Deposit(context.Context, string, int64, string, int64) (int, error) {
	return http.StatusServiceUnavailable, nil
}

func (c rejectingCluster) Withdraw(context.Context, string, int64, string, int64) (int, error) {
	return http.StatusBadGateway, nil
}

func TestGenerator_RejectedWriteIsNotUncertain(t *testing.T) {
	cluster, snapshot := newFakeCluster(10)
	ledger := loadedLedger(t, snapshot)
	observer := newCountingObserver()

	generator, err := NewGenerator(
		ledger, rejectingCluster{cluster}, observer, NewChecker(ledger, observer),
		[]string{"router-1"}, 1, defaultProfile(),
	)
	if err != nil {
		t.Fatal(err)
	}

	deposit := models.Operation{Kind: models.OperationDeposit, UserID: 1, Currency: "USD", Amount: 5, ScheduledAt: time.Now()}
	generator.write(context.Background(), "router-1", deposit)

	if ledger.Observe(1).Busy || ledger.Wallets(1)[0].Balance != 1000 {
		t.Fatal("503 means the write was not applied: no ledger change, no uncertainty")
	}

	withdraw := models.Operation{Kind: models.OperationWithdraw, UserID: 2, Currency: "USD", Amount: 5, ScheduledAt: time.Now()}
	generator.write(context.Background(), "router-1", withdraw)

	if !ledger.Observe(2).Busy {
		t.Fatal("502 means the node may have applied the write: user must become uncertain")
	}
}
