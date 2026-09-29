package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type RouterClient struct {
	http *http.Client
}

type userDTO struct {
	Wallets []struct {
		Currency string `json:"currency"`
		Balance  int64  `json:"balance"`
	} `json:"wallets"`
}

type operationDTO struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

func NewRouterClient(client *http.Client) *RouterClient {
	return &RouterClient{http: client}
}

func (c *RouterClient) Get(ctx context.Context, router string, userID int64) (models.UserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/wallet/%d", router, userID), nil)
	if err != nil {
		return models.UserResponse{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return models.UserResponse{}, err
	}
	defer drain(resp)

	result := models.UserResponse{Status: resp.StatusCode}

	if resp.StatusCode != http.StatusOK {
		return result, nil
	}

	var user userDTO
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return result, err
	}

	for _, wallet := range user.Wallets {
		result.Wallets = append(result.Wallets, models.Wallet{Currency: wallet.Currency, Balance: wallet.Balance})
	}

	return result, nil
}

func (c *RouterClient) Deposit(ctx context.Context, router string, userID int64, currency string, amount int64) (int, error) {
	return c.operate(ctx, router, userID, "deposit", currency, amount)
}

func (c *RouterClient) Withdraw(ctx context.Context, router string, userID int64, currency string, amount int64) (int, error) {
	return c.operate(ctx, router, userID, "withdraw", currency, amount)
}

func (c *RouterClient) operate(
	ctx context.Context,
	router string,
	userID int64,
	operation string,
	currency string,
	amount int64,
) (int, error) {
	body, err := json.Marshal(operationDTO{Currency: currency, Amount: amount})
	if err != nil {
		return 0, err
	}

	url := fmt.Sprintf("%s/wallet/%d/%s", router, userID, operation)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer drain(resp)

	return resp.StatusCode, nil
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
