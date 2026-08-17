package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
http:
  addr: ":18080"
auth:
  jwt_secret: "test-secret"
  access_token_ttl_seconds: 3600
  ws_token_ttl_seconds: 90
heartbeat:
  interval_seconds: 12
  timeout_seconds: 40
movement_persistence:
  interval_seconds: 4
  batch_size: 80
postgres:
  dsn: "postgres://demo:demo@127.0.0.1:5432/pocket_pet?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 11
  conn_max_lifetime_seconds: 1200
redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 1
  key_prefix: "pocket_pet_test"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadFromYAMLFile(path)
	if err != nil {
		t.Fatalf("LoadFromYAMLFile() error = %v", err)
	}
	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("cfg.HTTPAddr = %q, want %q", cfg.HTTPAddr, ":18080")
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("cfg.JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret")
	}
	if cfg.Postgres.DSN == "" {
		t.Fatal("cfg.Postgres.DSN = empty, want non-empty dsn")
	}
	if cfg.Redis.KeyPrefix != "pocket_pet_test" {
		t.Fatalf("cfg.Redis.KeyPrefix = %q, want %q", cfg.Redis.KeyPrefix, "pocket_pet_test")
	}
	if cfg.MovementPersistenceInterval.Seconds() != 4 {
		t.Fatalf("cfg.MovementPersistenceInterval = %v, want 4s", cfg.MovementPersistenceInterval)
	}
	if cfg.MovementPersistenceBatchSize != 80 {
		t.Fatalf("cfg.MovementPersistenceBatchSize = %d, want 80", cfg.MovementPersistenceBatchSize)
	}
}

// TestLoadFromYAMLFileUsesMovementPersistenceDefaults 验证旧配置未声明写回参数时仍使用受控默认值。
func TestLoadFromYAMLFileUsesMovementPersistenceDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
auth:
  jwt_secret: "test-secret"
heartbeat:
  interval_seconds: 10
  timeout_seconds: 30
postgres:
  dsn: "postgres://demo:demo@127.0.0.1:5432/pocket_pet?sslmode=disable"
redis:
  addr: "127.0.0.1:6379"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadFromYAMLFile(path)
	if err != nil {
		t.Fatalf("LoadFromYAMLFile() error = %v", err)
	}
	if cfg.MovementPersistenceInterval.Seconds() != defaultMovementPersistenceIntervalSeconds {
		t.Fatalf("cfg.MovementPersistenceInterval = %v, want %ds", cfg.MovementPersistenceInterval, defaultMovementPersistenceIntervalSeconds)
	}
	if cfg.MovementPersistenceBatchSize != defaultMovementPersistenceBatchSize {
		t.Fatalf("cfg.MovementPersistenceBatchSize = %d, want %d", cfg.MovementPersistenceBatchSize, defaultMovementPersistenceBatchSize)
	}
}

func TestLoadFromYAMLFileRejectsInvalidHeartbeat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
auth:
  jwt_secret: "test-secret"
heartbeat:
  interval_seconds: 30
  timeout_seconds: 20
postgres:
  dsn: "postgres://demo:demo@127.0.0.1:5432/pocket_pet?sslmode=disable"
redis:
  addr: "127.0.0.1:6379"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadFromYAMLFile(path)
	if err == nil {
		t.Fatal("LoadFromYAMLFile() error = nil, want validation failure")
	}
}
