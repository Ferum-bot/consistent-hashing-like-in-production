package main

import (
	"fmt"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/config"
)

type Config struct {
	Addr   string `yaml:"addr" env:"ADDR"`
	NodeID string `yaml:"node_id" env:"NODE_ID"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	if err := config.Load(path, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Addr == "" || cfg.NodeID == "" {
		return cfg, fmt.Errorf("%s: addr and node_id are required", path)
	}

	return cfg, nil
}
