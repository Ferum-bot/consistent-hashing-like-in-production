package api

import "github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"

type profileDTO struct {
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

type profilePatchDTO struct {
	Running        *bool    `json:"running"`
	RPS            *int     `json:"rps"`
	ReadRatio      *float64 `json:"read_ratio"`
	DepositShare   *float64 `json:"deposit_share"`
	ReadZipfS      *float64 `json:"read_zipf_s"`
	WriteZipfS     *float64 `json:"write_zipf_s"`
	HotShare       *float64 `json:"hot_share"`
	MerchantUserID *int64   `json:"merchant_user_id"`
	MaxAmount      *int64   `json:"max_amount"`
	VerifyRPS      *int     `json:"verify_rps"`
}

type ledgerDTO struct {
	Users     int   `json:"users"`
	Wallets   int   `json:"wallets"`
	Money     int64 `json:"money"`
	Uncertain int   `json:"uncertain_users"`
}

func toProfileDTO(profile models.Profile) profileDTO {
	return profileDTO{
		Running:        profile.Running,
		RPS:            profile.RPS,
		ReadRatio:      profile.ReadRatio,
		DepositShare:   profile.DepositShare,
		ReadZipfS:      profile.ReadZipfS,
		WriteZipfS:     profile.WriteZipfS,
		HotShare:       profile.HotShare,
		MerchantUserID: profile.MerchantUserID,
		MaxAmount:      profile.MaxAmount,
		VerifyRPS:      profile.VerifyRPS,
	}
}

func (d profilePatchDTO) applyTo(profile *models.Profile) {
	setIfPresent(&profile.Running, d.Running)
	setIfPresent(&profile.RPS, d.RPS)
	setIfPresent(&profile.ReadRatio, d.ReadRatio)
	setIfPresent(&profile.DepositShare, d.DepositShare)
	setIfPresent(&profile.ReadZipfS, d.ReadZipfS)
	setIfPresent(&profile.WriteZipfS, d.WriteZipfS)
	setIfPresent(&profile.HotShare, d.HotShare)
	setIfPresent(&profile.MerchantUserID, d.MerchantUserID)
	setIfPresent(&profile.MaxAmount, d.MaxAmount)
	setIfPresent(&profile.VerifyRPS, d.VerifyRPS)
}

func setIfPresent[T any](dst *T, value *T) {
	if value != nil {
		*dst = *value
	}
}

func toLedgerDTO(stats models.LedgerStats) ledgerDTO {
	return ledgerDTO{
		Users:     stats.Users,
		Wallets:   stats.Wallets,
		Money:     stats.Money,
		Uncertain: stats.Uncertain,
	}
}
