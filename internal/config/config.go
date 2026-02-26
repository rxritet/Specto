package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Host     string // SPECTO_HOST, default "0.0.0.0"
	Port     string // SPECTO_PORT, default "8080"
	LogLevel string // SPECTO_LOG_LEVEL, default "info"
}

// Load reads configuration from environment variables.
// Missing variables fall back to sensible defaults.
func Load() *Config {
	return &Config{
		Host:     envOr("SPECTO_HOST", "0.0.0.0"),
		Port:     envOr("SPECTO_PORT", "8080"),
		LogLevel: envOr("SPECTO_LOG_LEVEL", "info"),
	}
}

// Addr returns the "host:port" string suitable for net/http.Server.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// envOr returns the value of the environment variable named by key,
// or fallback if the variable is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
