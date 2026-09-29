package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/repositories"
)

func newService() *WalletService {
	return NewWalletService(repositories.NewMemoryRepository())
}

func TestWalletService_DepositOpensWallet(t *testing.T) {
	ctx := context.Background()
	svc := newService()

	if _, err := svc.Deposit(ctx, 1, "USD", 100); err != nil {
		t.Fatal(err)
	}
	wallet, err := svc.Deposit(ctx, 1, "USD", 50)
	if err != nil {
		t.Fatal(err)
	}
	if wallet != (models.Wallet{Currency: "USD", Balance: 150}) {
		t.Fatalf("wallet = %+v", wallet)
	}

	user, err := svc.User(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Wallets) != 1 {
		t.Fatalf("wallets = %+v, want one USD wallet", user.Wallets)
	}
}

func TestWalletService_DepositValidation(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		currency string
		amount   int64
	}{
		{"zero user id", 0, "USD", 1},
		{"lower-case currency", 1, "usd", 1},
		{"long currency", 1, "USDT", 1},
		{"zero amount", 1, "USD", 0},
		{"negative amount", 1, "USD", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newService().Deposit(context.Background(), tt.userID, tt.currency, tt.amount)
			if !errors.Is(err, models.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestWalletService_DepositOverflow(t *testing.T) {
	ctx := context.Background()
	svc := newService()
	if _, err := svc.Deposit(ctx, 1, "USD", math.MaxInt64); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Deposit(ctx, 1, "USD", 1)
	if !errors.Is(err, models.ErrBalanceOverflow) {
		t.Fatalf("err = %v, want ErrBalanceOverflow", err)
	}
}

func TestWalletService_ImportIsAllOrNothing(t *testing.T) {
	ctx := context.Background()
	svc := newService()

	err := svc.Import(ctx, []models.User{
		{ID: 1, Wallets: []models.Wallet{{Currency: "USD", Balance: 10}}},
		{ID: 2, Wallets: []models.Wallet{{Currency: "USD"}, {Currency: "USD"}}},
	})
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}

	if _, err := svc.User(ctx, 1); !errors.Is(err, models.ErrUserNotFound) {
		t.Fatalf("user 1 must not be imported, err = %v", err)
	}
}

func TestTopologyFence(t *testing.T) {
	fence := NewTopologyFence()

	steps := []struct {
		version int64
		want    int64
		wantErr error
	}{
		{version: 0, want: 0},
		{version: 3, want: 3},
		{version: 3, want: 3},
		{version: 2, want: 3, wantErr: models.ErrStaleTopology},
		{version: 5, want: 5},
	}
	for _, step := range steps {
		got, err := fence.Observe(step.version)
		if got != step.want || !errors.Is(err, step.wantErr) {
			t.Fatalf("Observe(%d) = %d, %v; want %d, %v", step.version, got, err, step.want, step.wantErr)
		}
	}
}

func TestWalletService_Withdraw(t *testing.T) {
	ctx := context.Background()
	svc := newService()

	if _, err := svc.Deposit(ctx, 1, "USD", 100); err != nil {
		t.Fatal(err)
	}

	wallet, err := svc.Withdraw(ctx, 1, "USD", 30)
	if err != nil {
		t.Fatal(err)
	}

	if wallet != (models.Wallet{Currency: "USD", Balance: 70}) {
		t.Fatalf("wallet = %+v", wallet)
	}
}

func TestWalletService_WithdrawErrors(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		currency string
		amount   int64
		wantErr  error
	}{
		{"more than balance", 1, "USD", 101, models.ErrInsufficientFunds},
		{"no wallet in currency", 1, "EUR", 1, models.ErrWalletNotFound},
		{"no user", 2, "USD", 1, models.ErrWalletNotFound},
		{"zero amount", 1, "USD", 0, models.ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svc := newService()

			if _, err := svc.Deposit(ctx, 1, "USD", 100); err != nil {
				t.Fatal(err)
			}

			_, err := svc.Withdraw(ctx, tt.userID, tt.currency, tt.amount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			user, _ := svc.User(ctx, 1)
			if user.Wallets[0].Balance != 100 || len(user.Wallets) != 1 {
				t.Fatalf("user changed after failed withdraw: %+v", user)
			}

			if _, err := svc.User(ctx, 2); !errors.Is(err, models.ErrUserNotFound) {
				t.Fatalf("failed withdraw must not create a user, err = %v", err)
			}
		})
	}
}

func TestTopologyFence_Strict(t *testing.T) {
	fence := NewTopologyFence()

	if fence.Strict() {
		t.Fatal("new fence must not be strict")
	}

	fence.SetStrict(true)

	if !fence.Strict() {
		t.Fatal("fence must become strict")
	}

	fence.SetStrict(false)

	if fence.Strict() {
		t.Fatal("fence must relax")
	}
}
