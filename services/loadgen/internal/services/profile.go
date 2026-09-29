package services

import "github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"

const (
	maxRPS       = 100_000
	maxVerifyRPS = 10_000
	maxAmount    = 1_000_000_000
)

func validateProfile(profile models.Profile) error {
	if profile.RPS < 0 || profile.RPS > maxRPS {
		return models.NewErrOutOfRange("rps", profile.RPS, 0, maxRPS)
	}

	if err := validateShare("read_ratio", profile.ReadRatio); err != nil {
		return err
	}

	if err := validateShare("deposit_share", profile.DepositShare); err != nil {
		return err
	}

	if err := validateShare("hot_share", profile.HotShare); err != nil {
		return err
	}

	if err := validateZipf("read_zipf_s", profile.ReadZipfS); err != nil {
		return err
	}

	if err := validateZipf("write_zipf_s", profile.WriteZipfS); err != nil {
		return err
	}

	if profile.MerchantUserID < 1 {
		return models.NewErrOutOfRange("merchant_user_id", profile.MerchantUserID, 1, "users")
	}

	if profile.MaxAmount < 1 || profile.MaxAmount > maxAmount {
		return models.NewErrOutOfRange("max_amount", profile.MaxAmount, 1, maxAmount)
	}

	if profile.VerifyRPS < 0 || profile.VerifyRPS > maxVerifyRPS {
		return models.NewErrOutOfRange("verify_rps", profile.VerifyRPS, 0, maxVerifyRPS)
	}

	return nil
}

func validateShare(name string, value float64) error {
	if value < 0 || value > 1 {
		return models.NewErrOutOfRange(name, value, 0, 1)
	}

	return nil
}

func validateZipf(name string, value float64) error {
	if value != 0 && value <= 1 {
		return models.NewErrZipfExponent(name, value)
	}

	return nil
}
