package api

import (
	"errors"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

func statusFor(err error) int {
	switch {
	case errors.Is(err, models.ErrInvalidArgument), errors.Is(err, request.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, models.ErrUserNotFound), errors.Is(err, models.ErrWalletNotFound):
		return http.StatusNotFound
	case errors.Is(err, models.ErrStaleTopology):
		return http.StatusConflict
	case errors.Is(err, models.ErrInsufficientFunds), errors.Is(err, models.ErrBalanceOverflow):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
