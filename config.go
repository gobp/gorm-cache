package cache

import (
	"time"

	"github.com/gofiber/storage/memory"

	"github.com/gobp/gorm-cache/serializers"
)

type Config struct {
	Storage Storage

	Prefix string

	Serializer Serializer

	Expires time.Duration

	KeyGenerator func(string) string
}

// ConfigDefault is the default config
var ConfigDefault = Config{
	Storage:      memory.New(),
	Serializer:   &serializers.JSONSerializer{},
	Prefix:       "gobp:cache:",
	KeyGenerator: func(s string) string { return s },
	Expires:      time.Hour,
}

// Helper function to set default values
func configDefault(config ...Config) Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.Storage == nil {
		cfg.Storage = ConfigDefault.Storage
	}
	if cfg.Serializer == nil {
		cfg.Serializer = ConfigDefault.Serializer
	}
	if cfg.Prefix == "" {
		cfg.Prefix = ConfigDefault.Prefix
	}
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = ConfigDefault.KeyGenerator
	}
	if int(cfg.Expires.Seconds()) == 0 {
		cfg.Expires = ConfigDefault.Expires
	}

	return cfg
}
