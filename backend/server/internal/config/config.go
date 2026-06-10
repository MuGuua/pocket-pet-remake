package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	KeyPrefix string
}

type Config struct {
	HTTPAddr          string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	WSTokenTTL        time.Duration
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	Postgres          PostgresConfig
	Redis             RedisConfig
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:          getString("PP_HTTP_ADDR", ":8080"),
		JWTSecret:         getString("PP_JWT_SECRET", "change-me"),
		AccessTokenTTL:    getSeconds("PP_ACCESS_TOKEN_TTL_SECONDS", 7200),
		WSTokenTTL:        getSeconds("PP_WS_TOKEN_TTL_SECONDS", 60),
		HeartbeatInterval: getSeconds("PP_HEARTBEAT_INTERVAL_SECONDS", 10),
		HeartbeatTimeout:  getSeconds("PP_HEARTBEAT_TIMEOUT_SECONDS", 30),
		Postgres: PostgresConfig{
			DSN:             getString("PP_POSTGRES_DSN", ""),
			MaxOpenConns:    getInt("PP_POSTGRES_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    getInt("PP_POSTGRES_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getSeconds("PP_POSTGRES_CONN_MAX_LIFETIME_SECONDS", 1800),
		},
		Redis: RedisConfig{
			Addr:      getString("PP_REDIS_ADDR", ""),
			Password:  getString("PP_REDIS_PASSWORD", ""),
			DB:        getInt("PP_REDIS_DB", 0),
			KeyPrefix: getString("PP_REDIS_KEY_PREFIX", "pocket_pet"),
		},
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("PP_JWT_SECRET must not be empty")
	}
	if cfg.HeartbeatTimeout <= cfg.HeartbeatInterval {
		return Config{}, fmt.Errorf("heartbeat timeout must be greater than heartbeat interval")
	}
	if cfg.Postgres.MaxOpenConns <= 0 {
		return Config{}, fmt.Errorf("PP_POSTGRES_MAX_OPEN_CONNS must be greater than zero")
	}
	if cfg.Postgres.MaxIdleConns < 0 {
		return Config{}, fmt.Errorf("PP_POSTGRES_MAX_IDLE_CONNS must not be negative")
	}
	if cfg.Redis.DB < 0 {
		return Config{}, fmt.Errorf("PP_REDIS_DB must not be negative")
	}
	if cfg.Postgres.DSN == "" {
		return Config{}, fmt.Errorf("PP_POSTGRES_DSN must not be empty")
	}
	if cfg.Redis.Addr == "" {
		return Config{}, fmt.Errorf("PP_REDIS_ADDR must not be empty")
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

func getInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
