package main

import (
	"fmt"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/config"
)

type Config struct {
	Addr            string        `yaml:"addr" env:"ADDR"`
	Name            string        `yaml:"name" env:"ROUTER_NAME"`
	Users           int64         `yaml:"users" env:"USERS"`
	SnapshotPath    string        `yaml:"snapshot_path" env:"SNAPSHOT_PATH"`
	GateTimeout     time.Duration `yaml:"gate_timeout" env:"GATE_TIMEOUT"`
	UpstreamTimeout time.Duration `yaml:"upstream_timeout" env:"UPSTREAM_TIMEOUT"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	if err := config.Load(path, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Addr == "" || cfg.Name == "" || cfg.SnapshotPath == "" {
		return cfg, fmt.Errorf("%s: addr, name and snapshot_path are required", path)
	}

	if cfg.Users < 1 {
		return cfg, fmt.Errorf("%s: users must be positive, got %d", path, cfg.Users)
	}

	if cfg.GateTimeout <= 0 || cfg.UpstreamTimeout <= 0 {
		return cfg, fmt.Errorf("%s: gate_timeout and upstream_timeout must be positive", path)
	}

	return cfg, nil
}
