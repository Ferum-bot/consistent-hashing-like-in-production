package models

import (
	"errors"
	"fmt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrBalanceOverflow   = errors.New("balance overflow")
	ErrStaleTopology     = errors.New("stale topology version")
)

func NewErrStaleVersion(current, got int64) error {
	return fmt.Errorf("%w: node is on %d, request has %d", ErrStaleTopology, current, got)
}

func NewErrTopologyVersionRequired() error {
	return fmt.Errorf("%w: topology version header is required", ErrStaleTopology)
}

func NewErrUserIDMustBePositive(userID int64) error {
	return fmt.Errorf("%w: user id must be positive, got %d", ErrInvalidArgument, userID)
}

func NewErrAmountMustBePositive(amount int64) error {
	return fmt.Errorf("%w: amount must be positive, got %d", ErrInvalidArgument, amount)
}

func NewErrCurrencyMustBeThreeLetters(currency string) error {
	return fmt.Errorf("%w: currency must be 3 letters, got %q", ErrInvalidArgument, currency)
}

func NewErrCurrencyMustBeUpperLatin(currency string) error {
	return fmt.Errorf("%w: currency must be upper-case latin letters, got %q", ErrInvalidArgument, currency)
}

func NewErrWalletNotFound(userID int64, currency string) error {
	return fmt.Errorf("%w: user %d has no %s wallet", ErrWalletNotFound, userID, currency)
}

func NewErrInsufficientFunds(balance, amount int64) error {
	return fmt.Errorf("%w: balance %d, requested %d", ErrInsufficientFunds, balance, amount)
}

func NewErrInvalidUser(userID int64, err error) error {
	return fmt.Errorf("user %d: %w", userID, err)
}

func NewErrDuplicateWallet(userID int64, currency string) error {
	return fmt.Errorf("%w: user %d has two %s wallets", ErrInvalidArgument, userID, currency)
}

func NewErrNegativeBalance(userID int64, currency string) error {
	return fmt.Errorf("%w: user %d has negative %s balance", ErrInvalidArgument, userID, currency)
}
