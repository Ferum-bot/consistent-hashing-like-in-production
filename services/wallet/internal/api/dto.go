package api

import "github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"

type userDTO struct {
	UserID  int64       `json:"user_id"`
	Wallets []walletDTO `json:"wallets"`
}

type walletDTO struct {
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
}

type operationRequest struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

type importResponse struct {
	Imported int `json:"imported"`
}

type fenceRequest struct {
	Strict bool `json:"strict"`
}

type fenceResponse struct {
	TopologyVersion int64 `json:"topology_version"`
	Strict          bool  `json:"strict"`
}

type evictRequest struct {
	UserIDs []int64 `json:"user_ids"`
}

type evictResponse struct {
	Evicted int `json:"evicted"`
}

type statsResponse struct {
	Node            string `json:"node"`
	TopologyVersion int64  `json:"topology_version"`
	Strict          bool   `json:"strict"`
	Users           int    `json:"users"`
	Wallets         int    `json:"wallets"`
}

func toUserDTO(user models.User) userDTO {
	wallets := make([]walletDTO, 0, len(user.Wallets))

	for _, wallet := range user.Wallets {
		wallets = append(wallets, toWalletDTO(wallet))
	}

	return userDTO{UserID: user.ID, Wallets: wallets}
}

func toWalletDTO(wallet models.Wallet) walletDTO {
	return walletDTO{Currency: wallet.Currency, Balance: wallet.Balance}
}

func (d userDTO) toModel() models.User {
	wallets := make([]models.Wallet, 0, len(d.Wallets))

	for _, wallet := range d.Wallets {
		wallets = append(wallets, models.Wallet{Currency: wallet.Currency, Balance: wallet.Balance})
	}

	return models.User{ID: d.UserID, Wallets: wallets}
}
