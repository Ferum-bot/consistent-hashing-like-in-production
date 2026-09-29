package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrNoTopology          = errors.New("topology is not set")
	ErrStaleTopology       = errors.New("stale topology version")
	ErrNodeUnavailable     = errors.New("node unavailable")
	ErrMigrationInProgress = errors.New("migration in progress")
	ErrNoMigration         = errors.New("no migration in progress")
	ErrMigrationTimeout    = errors.New("user is being migrated, retry later")
	ErrVersionMismatch     = errors.New("topology version mismatch")
	ErrTooManyDeferred     = errors.New("router has too many postponed changes")
)

func NewErrUnknownStrategy(name string) error {
	return fmt.Errorf("%w: unknown strategy %q", ErrInvalidArgument, name)
}

func NewErrVersionMustBePositive(version int64) error {
	return fmt.Errorf("%w: topology version must be positive, got %d", ErrInvalidArgument, version)
}

func NewErrVNodesOutOfRange(vnodes, limit int) error {
	return fmt.Errorf("%w: vnodes must be in [1, %d], got %d", ErrInvalidArgument, limit, vnodes)
}

func NewErrTooManyRingPoints(points, limit int) error {
	return fmt.Errorf("%w: ring would have %d points, limit is %d", ErrInvalidArgument, points, limit)
}

func NewErrTooManyDeferred(limit int) error {
	return fmt.Errorf("%w: limit is %d", ErrTooManyDeferred, limit)
}

func NewErrDelayOutOfRange(router string, delay, limit time.Duration) error {
	return fmt.Errorf("%w: membership delay of %s must be in [0, %s], got %s", ErrInvalidArgument, router, limit, delay)
}

func NewErrInvalidDelay(router, raw string) error {
	return fmt.Errorf("%w: membership delay of %s %q is not a duration", ErrInvalidArgument, router, raw)
}

func NewErrNoNodes() error {
	return fmt.Errorf("%w: topology has no nodes", ErrInvalidArgument)
}

func NewErrDuplicateNode(id string) error {
	return fmt.Errorf("%w: node %q is listed twice", ErrInvalidArgument, id)
}

func NewErrInvalidNode(id string, reason string) error {
	return fmt.Errorf("%w: node %q: %s", ErrInvalidArgument, id, reason)
}

func NewErrMigrationInProgress(version int64) error {
	return fmt.Errorf("%w: topology %d is not committed yet", ErrMigrationInProgress, version)
}

func NewErrPendingVersion(pending, got int64) error {
	return fmt.Errorf("%w: prepared topology is %d, got %d", ErrVersionMismatch, pending, got)
}

func NewErrMigrationTimeout(userID int64) error {
	return fmt.Errorf("%w: user %d", ErrMigrationTimeout, userID)
}

func NewErrStaleTopology(current, got int64) error {
	return fmt.Errorf("%w: current %d, got %d", ErrStaleTopology, current, got)
}

func NewErrNodeUnavailable(id string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrNodeUnavailable, id, err)
}

func NewErrNodeFenced(id string, reason string) error {
	reason = strings.TrimPrefix(reason, ErrStaleTopology.Error()+": ")

	return fmt.Errorf("%w: %s rejected the request: %s", ErrStaleTopology, id, reason)
}

func NewErrNodeRejected(id string, status int, reason string) error {
	return fmt.Errorf("%w: %s answered %d: %s", ErrNodeUnavailable, id, status, reason)
}

func NewErrReadSnapshot(err error) error {
	return fmt.Errorf("read snapshot: %w", err)
}

func NewErrSnapshotLine(line int, err error) error {
	return fmt.Errorf("snapshot line %d: %w", line, err)
}
