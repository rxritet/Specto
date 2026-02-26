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

	// Database
	DBProvider string // SPECTO_DB_PROVIDER: "postgres" | "bolt", default "bolt"
	DBDsn      string // SPECTO_DB_DSN (postgres connection string)
	DBPath     string // SPECTO_DB_PATH (bolt file path), default "specto.db"
}

// Load reads configuration from environment variables.
// Missing variables fall back to sensible defaults.
func Load() *Config {
	return &Config{
		Host:       envOr("SPECTO_HOST", "0.0.0.0"),
		Port:       envOr("SPECTO_PORT", "8080"),
		LogLevel:   envOr("SPECTO_LOG_LEVEL", "info"),
		DBProvider: envOr("SPECTO_DB_PROVIDER", "bolt"),
		DBDsn:      envOr("SPECTO_DB_DSN", ""),
		DBPath:     envOr("SPECTO_DB_PATH", "specto.db"),
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
