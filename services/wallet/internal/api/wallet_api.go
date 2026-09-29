package api

import (
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/response"
)

type WalletAPI struct {
	wallets WalletService
}

func NewWalletAPI(wallets WalletService) *WalletAPI {
	return &WalletAPI{wallets: wallets}
}

func (a *WalletAPI) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathInt64(r, "userID")
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	user, err := a.wallets.User(r.Context(), userID)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toUserDTO(user))
}

func (a *WalletAPI) Deposit(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathInt64(r, "userID")
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	var req operationRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	wallet, err := a.wallets.Deposit(r.Context(), userID, req.Currency, req.Amount)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toWalletDTO(wallet))
}

func (a *WalletAPI) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := request.PathInt64(r, "userID")
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	var req operationRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	wallet, err := a.wallets.Withdraw(r.Context(), userID, req.Currency, req.Amount)
	if err != nil {
		response.Error(w, statusFor(err), err)
		return
	}

	response.JSON(w, http.StatusOK, toWalletDTO(wallet))
}
