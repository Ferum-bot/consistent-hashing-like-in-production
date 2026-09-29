package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testConfig struct {
	Addr    string        `yaml:"addr" env:"TEST_ADDR"`
	Users   int64         `yaml:"users" env:"TEST_USERS"`
	Timeout time.Duration `yaml:"timeout" env:"TEST_TIMEOUT"`
	Targets []string      `yaml:"targets" env:"TEST_TARGETS"`
	Ratio   float64       `yaml:"ratio"`
	Nested  nestedConfig  `yaml:"nested"`
}

type nestedConfig struct {
	Running bool `yaml:"running" env:"TEST_RUNNING"`
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := writeConfig(t, "addr: \":8080\"\nusers: 10\ntimeout: 1s\ntargets: [a]\nratio: 0.5\n")

	t.Setenv("TEST_ADDR", ":9000")
	t.Setenv("TEST_USERS", "42")
	t.Setenv("TEST_TIMEOUT", "3s")
	t.Setenv("TEST_TARGETS", " b , c ,")

	var cfg testConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Addr != ":9000" || cfg.Users != 42 || cfg.Timeout != 3*time.Second || cfg.Ratio != 0.5 {
		t.Fatalf("cfg = %+v", cfg)
	}

	if len(cfg.Targets) != 2 || cfg.Targets[0] != "b" || cfg.Targets[1] != "c" {
		t.Fatalf("targets = %q", cfg.Targets)
	}
}

func TestLoad_RejectsMistakes(t *testing.T) {
	var cfg testConfig

	if err := Load(writeConfig(t, "adr: \":8080\"\n"), &cfg); err == nil {
		t.Fatal("unknown field in the file must be an error")
	}

	t.Setenv("TEST_USERS", "many")

	if err := Load(writeConfig(t, "users: 1\n"), &cfg); err == nil {
		t.Fatal("wrong value in the environment must be an error")
	}
}

func TestLoad_NestedAndEmpty(t *testing.T) {
	t.Setenv("TEST_RUNNING", "true")
	t.Setenv("TEST_ADDR", ":7000")

	var cfg testConfig
	if err := Load(writeConfig(t, "# only env\n"), &cfg); err != nil {
		t.Fatal(err)
	}

	if !cfg.Nested.Running || cfg.Addr != ":7000" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoad_RejectsUnsupportedEnvType(t *testing.T) {
	var cfg struct {
		Port uint `yaml:"port" env:"TEST_PORT"`
	}

	if err := Load(writeConfig(t, "port: 1\n"), &cfg); err == nil {
		t.Fatal("an env tag on an unsupported type must be an error even without the variable")
	}
}
