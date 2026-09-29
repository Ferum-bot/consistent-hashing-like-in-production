package services

import (
	"context"
	"math"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type WalletService struct {
	repo Repository
}

func NewWalletService(repo Repository) *WalletService {
	return &WalletService{repo: repo}
}

func (s *WalletService) User(ctx context.Context, id int64) (models.User, error) {
	return s.repo.Get(ctx, id)
}

func (s *WalletService) Deposit(
	ctx context.Context,
	userID int64,
	currency string,
	amount int64,
) (models.Wallet, error) {
	if err := validateOperation(userID, currency, amount); err != nil {
		return models.Wallet{}, err
	}

	var deposited models.Wallet

	err := s.repo.Update(ctx, userID, func(user *models.User) error {
		wallet := user.OpenWallet(currency)

		if wallet.Balance > math.MaxInt64-amount {
			return models.ErrBalanceOverflow
		}

		wallet.Balance += amount
		deposited = *wallet

		return nil
	})
	if err != nil {
		return models.Wallet{}, err
	}

	return deposited, nil
}

func (s *WalletService) Withdraw(
	ctx context.Context,
	userID int64,
	currency string,
	amount int64,
) (models.Wallet, error) {
	if err := validateOperation(userID, currency, amount); err != nil {
		return models.Wallet{}, err
	}

	var withdrawn models.Wallet

	err := s.repo.Update(ctx, userID, func(user *models.User) error {
		wallet := user.FindWallet(currency)

		if wallet == nil {
			return models.NewErrWalletNotFound(userID, currency)
		}

		if wallet.Balance < amount {
			return models.NewErrInsufficientFunds(wallet.Balance, amount)
		}

		wallet.Balance -= amount
		withdrawn = *wallet

		return nil
	})
	if err != nil {
		return models.Wallet{}, err
	}

	return withdrawn, nil
}

func (s *WalletService) Evict(ctx context.Context, ids []int64) (int, error) {
	return s.repo.DeleteMany(ctx, ids)
}

func (s *WalletService) Export(ctx context.Context) ([]models.User, error) {
	return s.repo.Snapshot(ctx)
}

func (s *WalletService) Import(ctx context.Context, users []models.User) error {
	for _, user := range users {
		if err := validateUser(user); err != nil {
			return err
		}
	}

	return s.repo.Save(ctx, users)
}

func (s *WalletService) Stats(ctx context.Context) (models.Stats, error) {
	return s.repo.Stats(ctx)
}
