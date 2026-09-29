package services

import (
	"math/rand/v2"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type picker struct {
	random    *rand.Rand
	profile   models.Profile
	users     int
	readZipf  *rand.Zipf
	writeZipf *rand.Zipf
}

func newPicker(random *rand.Rand, profile models.Profile, users int) *picker {
	return &picker{
		random:    random,
		profile:   profile,
		users:     users,
		readZipf:  newZipf(random, profile.ReadZipfS, users),
		writeZipf: newZipf(random, profile.WriteZipfS, users),
	}
}

func (p *picker) next(ledger Ledger, scheduledAt time.Time) models.Operation {
	if p.random.Float64() < p.profile.ReadRatio {
		return models.Operation{
			Kind:        models.OperationRead,
			UserID:      p.user(p.readZipf),
			ScheduledAt: scheduledAt,
		}
	}

	userID := p.writeUser()

	wallets := ledger.Wallets(userID)
	if len(wallets) == 0 {
		return models.Operation{Kind: models.OperationRead, UserID: userID, ScheduledAt: scheduledAt}
	}

	kind := models.OperationWithdraw

	if p.random.Float64() < p.profile.DepositShare {
		kind = models.OperationDeposit
	}

	return models.Operation{
		Kind:        kind,
		UserID:      userID,
		Currency:    p.currency(wallets),
		Amount:      1 + p.random.Int64N(p.profile.MaxAmount),
		ScheduledAt: scheduledAt,
	}
}

func (p *picker) writeUser() int64 {
	merchantKnown := p.profile.MerchantUserID <= int64(p.users)

	if merchantKnown && p.random.Float64() < p.profile.HotShare {
		return p.profile.MerchantUserID
	}

	return p.user(p.writeZipf)
}

func (p *picker) user(zipf *rand.Zipf) int64 {
	if zipf == nil {
		return 1 + p.random.Int64N(int64(p.users))
	}

	return 1 + int64(zipf.Uint64())
}

func (p *picker) currency(wallets []models.Wallet) string {
	return wallets[p.random.IntN(len(wallets))].Currency
}

func newZipf(random *rand.Rand, s float64, users int) *rand.Zipf {
	if s == 0 || users < 2 {
		return nil
	}

	return rand.NewZipf(random, s, 1, uint64(users-1))
}
