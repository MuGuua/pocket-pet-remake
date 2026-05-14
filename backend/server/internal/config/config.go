package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr          string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	WSTokenTTL        time.Duration
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	DemoAccount       string
	DemoPassword      string
	DemoPlayerName    string
	DemoAccountID     uint64
	DemoPlayerID      uint64
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:          getString("PP_HTTP_ADDR", ":8080"),
		JWTSecret:         getString("PP_JWT_SECRET", "change-me"),
		AccessTokenTTL:    getSeconds("PP_ACCESS_TOKEN_TTL_SECONDS", 7200),
		WSTokenTTL:        getSeconds("PP_WS_TOKEN_TTL_SECONDS", 60),
		HeartbeatInterval: getSeconds("PP_HEARTBEAT_INTERVAL_SECONDS", 10),
		HeartbeatTimeout:  getSeconds("PP_HEARTBEAT_TIMEOUT_SECONDS", 30),
		DemoAccount:       getString("PP_DEMO_ACCOUNT", "demo"),
		DemoPassword:      getString("PP_DEMO_PASSWORD", "demo123"),
		DemoPlayerName:    getString("PP_DEMO_PLAYER_NAME", "DemoTrainer"),
	}

	var err error
	cfg.DemoAccountID, err = getUint64("PP_DEMO_ACCOUNT_ID", 1)
	if err != nil {
		return Config{}, err
	}

	cfg.DemoPlayerID, err = getUint64("PP_DEMO_PLAYER_ID", 10001)
	if err != nil {
		return Config{}, err
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("PP_JWT_SECRET must not be empty")
	}
	if cfg.HeartbeatTimeout <= cfg.HeartbeatInterval {
		return Config{}, fmt.Errorf("heartbeat timeout must be greater than heartbeat interval")
	}

	return cfg, nil
}

func getString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getSeconds(key string, defaultSeconds int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(defaultSeconds) * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(defaultSeconds) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func getUint64(key string, defaultValue uint64) (uint64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
