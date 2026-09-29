package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const reloadTimeout = 5 * time.Minute

type Profile struct {
	Running        bool    `json:"running"`
	RPS            int     `json:"rps"`
	ReadRatio      float64 `json:"read_ratio"`
	DepositShare   float64 `json:"deposit_share"`
	ReadZipfS      float64 `json:"read_zipf_s"`
	WriteZipfS     float64 `json:"write_zipf_s"`
	HotShare       float64 `json:"hot_share"`
	MerchantUserID int64   `json:"merchant_user_id"`
	MaxAmount      int64   `json:"max_amount"`
	VerifyRPS      int     `json:"verify_rps"`
}

type Ledger struct {
	Users     int   `json:"users"`
	Wallets   int   `json:"wallets"`
	Money     int64 `json:"money"`
	Uncertain int   `json:"uncertain_users"`
}

type Loadgen struct {
	endpoint
}

func NewLoadgen(url string, client *http.Client) *Loadgen {
	return &Loadgen{endpoint: newEndpoint("loadgen", url, client)}
}

func (l *Loadgen) Profile(ctx context.Context) (Profile, error) {
	var profile Profile

	err := l.get(ctx, "/config", &profile)

	return profile, err
}

func (l *Loadgen) UpdateProfile(ctx context.Context, patch json.RawMessage) (Profile, error) {
	var profile Profile

	err := l.send(ctx, http.MethodPatch, "/config", patch, quickTimeout, &profile)

	return profile, err
}

func (l *Loadgen) Ledger(ctx context.Context) (Ledger, bool, error) {
	var ledger Ledger

	err := l.get(ctx, "/ledger", &ledger)
	if HasStatus(err, http.StatusServiceUnavailable) {
		return Ledger{}, false, nil
	}

	if err != nil {
		return Ledger{}, false, err
	}

	return ledger, true, nil
}

func (l *Loadgen) ReloadLedger(ctx context.Context) (Ledger, error) {
	var ledger Ledger

	err := l.send(ctx, http.MethodPost, "/ledger/reload", nil, reloadTimeout, &ledger)

	return ledger, err
}
