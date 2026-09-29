package api

import (
	"errors"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

func statusFor(err error) int {
	switch {
	case errors.Is(err, models.ErrInvalidArgument), errors.Is(err, request.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, models.ErrLedgerNotLoaded):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
