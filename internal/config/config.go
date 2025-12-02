package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server    ServerConfig      `yaml:"server"`
	Reindexer ReindexerConfig   `yaml:"reindexer"`
	Cache     CacheConfig       `yaml:"cache"`
	App       ApplicationConfig `yaml:"app"`
}

type ServerConfig struct {
	Port         int           `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" env-default:"60s"`
}

type ReindexerConfig struct {
	DSN       string `yaml:"dsn" env:"REINDEXER_DSN" env-required:"true"`
	Namespace string `yaml:"namespace" env:"REINDEXER_NAMESPACE" env-default:"documents"`
}

type CacheConfig struct {
	TTL             time.Duration `yaml:"ttl" env:"CACHE_TTL" env-default:"15m"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" env:"CACHE_CLEANUP_INTERVAL" env-default:"30m"`
	Capacity        int           `yaml:"capacity" env:"CACHE_CAPACITY" env-default:"1000"`
}

type ApplicationConfig struct {
	Env      string `yaml:"env" env:"APP_ENV" env-default:"development"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	if path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Если файла нет, пробуем читать только из ENV
			if err := cleanenv.ReadEnv(cfg); err != nil {
				return nil, fmt.Errorf("failed to read env config: %w", err)
			}
			return cfg, nil
		}

		if err := cleanenv.ReadConfig(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, fmt.Errorf("failed to read env config: %w", err)
		}
	}

	return cfg, nil
}
