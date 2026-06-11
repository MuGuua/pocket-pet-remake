package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
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

type yamlConfig struct {
	HTTP struct {
		Addr string `yaml:"addr"`
	} `yaml:"http"`
	Auth struct {
		JWTSecret             string `yaml:"jwt_secret"`
		AccessTokenTTLSeconds int    `yaml:"access_token_ttl_seconds"`
		WSTokenTTLSeconds     int    `yaml:"ws_token_ttl_seconds"`
	} `yaml:"auth"`
	Heartbeat struct {
		IntervalSeconds int `yaml:"interval_seconds"`
		TimeoutSeconds  int `yaml:"timeout_seconds"`
	} `yaml:"heartbeat"`
	Postgres struct {
		DSN                    string `yaml:"dsn"`
		MaxOpenConns           int    `yaml:"max_open_conns"`
		MaxIdleConns           int    `yaml:"max_idle_conns"`
		ConnMaxLifetimeSeconds int    `yaml:"conn_max_lifetime_seconds"`
	} `yaml:"postgres"`
	Redis struct {
		Addr      string `yaml:"addr"`
		Password  string `yaml:"password"`
		DB        int    `yaml:"db"`
		KeyPrefix string `yaml:"key_prefix"`
	} `yaml:"redis"`
}

// LoadFromYAMLFile reads one YAML config file and translates it into the
// runtime-facing Config shape that the rest of the service already depends on.
func LoadFromYAMLFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read yaml config %s: %w", path, err)
	}

	var raw yamlConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse yaml config %s: %w", path, err)
	}

	cfg := Config{
		HTTPAddr:          stringOrDefault(raw.HTTP.Addr, ":8080"),
		JWTSecret:         stringOrDefault(raw.Auth.JWTSecret, "change-me"),
		AccessTokenTTL:    secondsOrDefault(raw.Auth.AccessTokenTTLSeconds, 7200),
		WSTokenTTL:        secondsOrDefault(raw.Auth.WSTokenTTLSeconds, 60),
		HeartbeatInterval: secondsOrDefault(raw.Heartbeat.IntervalSeconds, 10),
		HeartbeatTimeout:  secondsOrDefault(raw.Heartbeat.TimeoutSeconds, 30),
		Postgres: PostgresConfig{
			DSN:             raw.Postgres.DSN,
			MaxOpenConns:    intOrDefault(raw.Postgres.MaxOpenConns, 20),
			MaxIdleConns:    intOrDefault(raw.Postgres.MaxIdleConns, 10),
			ConnMaxLifetime: secondsOrDefault(raw.Postgres.ConnMaxLifetimeSeconds, 1800),
		},
		Redis: RedisConfig{
			Addr:      raw.Redis.Addr,
			Password:  raw.Redis.Password,
			DB:        raw.Redis.DB,
			KeyPrefix: stringOrDefault(raw.Redis.KeyPrefix, "pocket_pet"),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret must not be empty")
	}
	if c.HeartbeatTimeout <= c.HeartbeatInterval {
		return fmt.Errorf("heartbeat.timeout_seconds must be greater than heartbeat.interval_seconds")
	}
	if c.Postgres.MaxOpenConns <= 0 {
		return fmt.Errorf("postgres.max_open_conns must be greater than zero")
	}
	if c.Postgres.MaxIdleConns < 0 {
		return fmt.Errorf("postgres.max_idle_conns must not be negative")
	}
	if c.Redis.DB < 0 {
		return fmt.Errorf("redis.db must not be negative")
	}
	if c.Postgres.DSN == "" {
		return fmt.Errorf("postgres.dsn must not be empty")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr must not be empty")
	}
	return nil
}

func stringOrDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func secondsOrDefault(value int, defaultSeconds int) time.Duration {
	if value <= 0 {
		value = defaultSeconds
	}
	return time.Duration(value) * time.Second
}

func intOrDefault(value int, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}
