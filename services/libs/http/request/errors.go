package request

import (
	"errors"
	"fmt"
)

var ErrInvalid = errors.New("invalid request")

func NewErrNotANumber(name, raw string) error {
	return fmt.Errorf("%w: %s %q is not a number", ErrInvalid, name, raw)
}

func NewErrDecodeBody(err error) error {
	return fmt.Errorf("%w: decode body: %v", ErrInvalid, err)
}

func NewErrDecodeLine(line int, err error) error {
	return fmt.Errorf("%w: decode line %d: %v", ErrInvalid, line, err)
}
