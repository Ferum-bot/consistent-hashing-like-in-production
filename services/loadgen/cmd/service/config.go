package main

import (
	"fmt"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/config"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type Config struct {
	Addr           string        `yaml:"addr" env:"ADDR"`
	Targets        []string      `yaml:"targets" env:"TARGETS"`
	SnapshotPath   string        `yaml:"snapshot_path" env:"SNAPSHOT_PATH"`
	Workers        int           `yaml:"workers" env:"WORKERS"`
	RequestTimeout time.Duration `yaml:"request_timeout" env:"REQUEST_TIMEOUT"`
	Profile        ProfileConfig `yaml:"profile"`
}

type ProfileConfig struct {
	Running        bool    `yaml:"running"`
	RPS            int     `yaml:"rps"`
	ReadRatio      float64 `yaml:"read_ratio"`
	DepositShare   float64 `yaml:"deposit_share"`
	ReadZipfS      float64 `yaml:"read_zipf_s"`
	WriteZipfS     float64 `yaml:"write_zipf_s"`
	HotShare       float64 `yaml:"hot_share"`
	MerchantUserID int64   `yaml:"merchant_user_id"`
	MaxAmount      int64   `yaml:"max_amount"`
	VerifyRPS      int     `yaml:"verify_rps"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	if err := config.Load(path, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Addr == "" || cfg.SnapshotPath == "" {
		return cfg, fmt.Errorf("%s: addr and snapshot_path are required", path)
	}

	if len(cfg.Targets) == 0 {
		return cfg, fmt.Errorf("%s: targets need at least one router", path)
	}

	if cfg.Workers < 1 {
		return cfg, fmt.Errorf("%s: workers must be positive, got %d", path, cfg.Workers)
	}

	if cfg.RequestTimeout <= 0 {
		return cfg, fmt.Errorf("%s: request_timeout must be positive, got %s", path, cfg.RequestTimeout)
	}

	return cfg, nil
}

func (c ProfileConfig) toModel() models.Profile {
	return models.Profile{
		Running:        c.Running,
		RPS:            c.RPS,
		ReadRatio:      c.ReadRatio,
		DepositShare:   c.DepositShare,
		ReadZipfS:      c.ReadZipfS,
		WriteZipfS:     c.WriteZipfS,
		HotShare:       c.HotShare,
		MerchantUserID: c.MerchantUserID,
		MaxAmount:      c.MaxAmount,
		VerifyRPS:      c.VerifyRPS,
	}
}
