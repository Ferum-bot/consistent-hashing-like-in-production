package services

import "github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"

func validateUserID(id int64) error {
	if id <= 0 {
		return models.NewErrUserIDMustBePositive(id)
	}

	return nil
}

func validateCurrency(currency string) error {
	if len(currency) != 3 {
		return models.NewErrCurrencyMustBeThreeLetters(currency)
	}

	for _, c := range currency {
		if c < 'A' || c > 'Z' {
			return models.NewErrCurrencyMustBeUpperLatin(currency)
		}
	}

	return nil
}

func validateOperation(userID int64, currency string, amount int64) error {
	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := validateCurrency(currency); err != nil {
		return err
	}

	if amount <= 0 {
		return models.NewErrAmountMustBePositive(amount)
	}

	return nil
}

func validateUser(user models.User) error {
	if err := validateUserID(user.ID); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(user.Wallets))

	for _, wallet := range user.Wallets {
		if err := validateCurrency(wallet.Currency); err != nil {
			return models.NewErrInvalidUser(user.ID, err)
		}

		if _, ok := seen[wallet.Currency]; ok {
			return models.NewErrDuplicateWallet(user.ID, wallet.Currency)
		}

		seen[wallet.Currency] = struct{}{}

		if wallet.Balance < 0 {
			return models.NewErrNegativeBalance(user.ID, wallet.Currency)
		}
	}

	return nil
}
