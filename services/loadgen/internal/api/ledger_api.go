package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type LedgerAPI struct {
	ledger LedgerService
}

func NewLedgerAPI(ledger LedgerService) *LedgerAPI {
	return &LedgerAPI{ledger: ledger}
}

func (a *LedgerAPI) Get(w http.ResponseWriter, r *http.Request) {
	stats, err := a.ledger.Stats(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toLedgerDTO(stats))
}

func (a *LedgerAPI) Reload(w http.ResponseWriter, r *http.Request) {
	stats, err := a.ledger.Reload(r.Context())
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toLedgerDTO(stats))
}
