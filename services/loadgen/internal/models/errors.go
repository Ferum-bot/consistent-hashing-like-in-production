package models

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrLedgerNotLoaded   = errors.New("ledger is not loaded")
	ErrSnapshotNotSorted = errors.New("snapshot must list users 1..N in order")
)

func NewErrOutOfRange(name string, value any, low, high any) error {
	return fmt.Errorf("%w: %s must be in [%v, %v], got %v", ErrInvalidArgument, name, low, high, value)
}

func NewErrZipfExponent(name string, value float64) error {
	return fmt.Errorf("%w: %s must be 0 or greater than 1, got %v", ErrInvalidArgument, name, value)
}

func NewErrReadSnapshot(err error) error {
	return fmt.Errorf("read snapshot: %w", err)
}

func NewErrSnapshotLine(line int, err error) error {
	return fmt.Errorf("snapshot line %d: %w", line, err)
}

func NewErrSnapshotOrder(line int, userID int64) error {
	return fmt.Errorf("%w: line %d has user_id %d", ErrSnapshotNotSorted, line, userID)
}

func NewErrTooManyCurrencies(currency string) error {
	return fmt.Errorf("snapshot has too many currencies, %q does not fit", currency)
}
