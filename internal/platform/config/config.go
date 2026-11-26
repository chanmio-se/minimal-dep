// Package config loads runtime configuration from environment variables.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the complete runtime configuration of the application.
type Config struct {
	Addr            string
	Env             string
	LogLevel        string
	DatabaseDriver  string
	DatabaseDSN     string
	SessionSecret   string
	SessionTTL      time.Duration
	MaxBodyBytes    int64
	SecureCookies   bool
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// Load reads configuration from the environment and fills in development defaults.
func Load() (Config, error) {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return Config{}, fmt.Errorf("generate session secret: %w", err)
		}
		secret = hex.EncodeToString(buf)
	}

	requestTimeout, err := durationEnv("REQUEST_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := durationEnv("SESSION_TTL", 8*time.Hour)
	if err != nil {
		return Config{}, err
	}
	maxBodyBytes, err := int64Env("MAX_BODY_BYTES", 1<<20)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Addr:            envDefault("ADDR", ":8080"),
		Env:             envDefault("APP_ENV", "development"),
		LogLevel:        envDefault("LOG_LEVEL", "info"),
		DatabaseDriver:  os.Getenv("DB_DRIVER"),
		DatabaseDSN:     os.Getenv("DB_DSN"),
		SessionSecret:   secret,
		SessionTTL:      sessionTTL,
		MaxBodyBytes:    maxBodyBytes,
		SecureCookies:   boolEnv("SECURE_COOKIES", false),
		RequestTimeout:  requestTimeout,
		ShutdownTimeout: shutdownTimeout,
	}
	return cfg, cfg.Validate()
}

// Validate checks that the configuration is usable.
func (c Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("addr is required")
	}
	if c.RequestTimeout <= 0 || c.ShutdownTimeout <= 0 || c.SessionTTL <= 0 {
		return fmt.Errorf("timeouts must be positive")
	}
	if c.MaxBodyBytes <= 0 {
		return fmt.Errorf("max body bytes must be positive")
	}
	if len(c.SessionSecret) < 16 {
		return fmt.Errorf("session secret must be at least 16 characters")
	}
	if (c.DatabaseDriver == "") != (c.DatabaseDSN == "") {
		return fmt.Errorf("DB_DRIVER and DB_DSN must be set together")
	}
	return nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
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

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return d, nil
}

func int64Env(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return n, nil
}
