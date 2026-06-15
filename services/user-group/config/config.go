// Package config loads and validates gateway configuration.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort     string        `envconfig:"PORT" default:"8090"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	ReadTimeout    time.Duration `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout   time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout    time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`

	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"5432"`
	DBName     string `envconfig:"DB_NAME" default:"user_group"`
	DBUser     string `envconfig:"DB_USER" default:"user_group"`
	DBPassword string `envconfig:"DB_PASSWORD" default:"changeme"`
}

func Load() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
