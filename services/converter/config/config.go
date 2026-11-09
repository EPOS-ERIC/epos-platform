// Package config loads converter configuration.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort   string        `envconfig:"SERVER_PORT" default:"8200"`
	DBHost       string        `envconfig:"DB_HOST" default:"localhost"`
	DBPort       string        `envconfig:"DB_PORT" default:"5432"`
	DBName       string        `envconfig:"DB_NAME" default:"converter"`
	DBUser       string        `envconfig:"DB_USER" default:"converter"`
	DBPassword   string        `envconfig:"DB_PASSWORD" default:"converter"`
	PluginsPath  string        `envconfig:"PLUGINS_PATH" default:"./plugins"`
	MaxBytes     int64         `envconfig:"MAX_PAYLOAD_BYTES" default:"1073741824"`
	Timeout      time.Duration `envconfig:"EXECUTION_TIMEOUT" default:"30s"`
	SyncInterval time.Duration `envconfig:"SYNC_INTERVAL" default:"5m"`
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	if cfg.ServerPort == "" || cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBName == "" || cfg.DBUser == "" || cfg.DBPassword == "" || cfg.PluginsPath == "" {
		return Config{}, fmt.Errorf("server, database, and plugin path configuration must not be empty")
	}
	if cfg.MaxBytes <= 0 || cfg.Timeout <= 0 || cfg.SyncInterval <= 0 {
		return Config{}, fmt.Errorf("payload limit, execution timeout, and sync interval must be positive")
	}
	return cfg, nil
}
