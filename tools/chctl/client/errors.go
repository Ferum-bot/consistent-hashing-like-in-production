package client

import (
	"context"
	"errors"
	"fmt"
	"syscall"
)

type UnreachableError struct {
	Service string
	URL     string
	Err     error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("%s %s is not reachable: %s", e.Service, e.URL, e.Reason())
}

func (e *UnreachableError) Unwrap() error {
	return e.Err
}

func (e *UnreachableError) Reason() string {
	switch {
	case errors.Is(e.Err, syscall.ECONNREFUSED):
		return "nothing listens on this address"
	case errors.Is(e.Err, context.DeadlineExceeded):
		return "no answer in time"
	default:
		return e.Err.Error()
	}
}

type ResponseError struct {
	Service string
	URL     string
	Status  int
	Message string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("%s %s answered %d: %s", e.Service, e.URL, e.Status, e.Message)
}

func HasStatus(err error, status int) bool {
	var response *ResponseError

	return errors.As(err, &response) && response.Status == status
}

func IsUnreachable(err error) bool {
	var unreachable *UnreachableError

	return errors.As(err, &unreachable)
}
