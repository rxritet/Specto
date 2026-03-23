package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Host     string // SPECTO_HOST, default "0.0.0.0"
	Port     string // SPECTO_PORT, default "8080"
	LogLevel string // SPECTO_LOG_LEVEL, default "info"

	// Storage (all are required in triple-store mode)
	PostgresDSN string // SPECTO_POSTGRES_DSN (postgres connection string)
	RedisAddr   string // SPECTO_REDIS_ADDR (host:port), default "localhost:6379"
	RedisPass   string // SPECTO_REDIS_PASSWORD
	RedisDB     int    // SPECTO_REDIS_DB, default 0
	BoltPath    string // SPECTO_BOLT_PATH (bolt file path), default "specto-audit.db"

	// Redis behavior
	RateLimitPerMinute int           // SPECTO_RATE_LIMIT_PER_MINUTE, default 60
	BalanceCacheTTL    time.Duration // SPECTO_BALANCE_CACHE_TTL, default 30s

	// Authentication
	AuthSecret        string        // SPECTO_AUTH_SECRET, default dev-only secret
	AuthSessionTTL    time.Duration // SPECTO_AUTH_SESSION_TTL, default 24h
	AuthSecureCookies bool          // SPECTO_AUTH_SECURE_COOKIES, default false
}

// Load reads configuration from environment variables.
// Missing variables fall back to sensible defaults.
func Load() *Config {
	return &Config{
		Host:               envOr("SPECTO_HOST", "0.0.0.0"),
		Port:               envOr("SPECTO_PORT", "8080"),
		LogLevel:           envOr("SPECTO_LOG_LEVEL", "info"),
		PostgresDSN:        envOr("SPECTO_POSTGRES_DSN", "postgres://user:password@localhost:5432/bank_db?sslmode=disable"),
		RedisAddr:          envOr("SPECTO_REDIS_ADDR", "localhost:6379"),
		RedisPass:          envOr("SPECTO_REDIS_PASSWORD", ""),
		RedisDB:            envIntOr("SPECTO_REDIS_DB", 0),
		BoltPath:           envOr("SPECTO_BOLT_PATH", "specto-audit.db"),
		RateLimitPerMinute: envIntOr("SPECTO_RATE_LIMIT_PER_MINUTE", 60),
		BalanceCacheTTL:    envDurationOr("SPECTO_BALANCE_CACHE_TTL", 30*time.Second),
		AuthSecret:         envOr("SPECTO_AUTH_SECRET", "specto-dev-secret-change-me"),
		AuthSessionTTL:     envDurationOr("SPECTO_AUTH_SESSION_TTL", 24*time.Hour),
		AuthSecureCookies:  envBoolOr("SPECTO_AUTH_SECURE_COOKIES", false),
	}
}

// Validate verifies required configuration for triple-store operation.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.PostgresDSN) == "" {
		return fmt.Errorf("SPECTO_POSTGRES_DSN is required")
	}
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("SPECTO_REDIS_ADDR is required")
	}
	if strings.TrimSpace(c.BoltPath) == "" {
		return fmt.Errorf("SPECTO_BOLT_PATH is required")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("SPECTO_RATE_LIMIT_PER_MINUTE must be > 0")
	}
	if c.BalanceCacheTTL <= 0 {
		return fmt.Errorf("SPECTO_BALANCE_CACHE_TTL must be > 0")
	}
	if c.AuthSessionTTL <= 0 {
		return fmt.Errorf("SPECTO_AUTH_SESSION_TTL must be > 0")
	}
	if strings.TrimSpace(c.AuthSecret) == "" {
		return fmt.Errorf("SPECTO_AUTH_SECRET is required")
	}
	return nil
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

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envBoolOr(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
