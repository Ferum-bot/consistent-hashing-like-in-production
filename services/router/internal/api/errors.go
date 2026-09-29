package api

import (
	"errors"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

func statusFor(err error) int {
	switch {
	case errors.Is(err, models.ErrInvalidArgument), errors.Is(err, request.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, models.ErrStaleTopology),
		errors.Is(err, models.ErrMigrationInProgress),
		errors.Is(err, models.ErrNoMigration),
		errors.Is(err, models.ErrVersionMismatch):
		return http.StatusConflict
	case errors.Is(err, models.ErrTooManyDeferred):
		return http.StatusServiceUnavailable
	case errors.Is(err, models.ErrMigrationTimeout):
		return http.StatusServiceUnavailable
	case errors.Is(err, models.ErrNodeUnavailable):
		return http.StatusBadGateway
	case errors.Is(err, models.ErrNoTopology):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
