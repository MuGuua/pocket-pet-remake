package postgres

import (
	"context"
	"os"
	"testing"

	"pocket-pet-remake/server/internal/module/world"
)

// 在独立本地 schema 执行正式迁移，验证默认关闭、审计和配置持久化，不操作正式数据库。
func TestWorldAOIConfigMigrationPostgres(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/131_world_aoi_config.sql")
	if err != nil {
		t.Fatal(err)
	}
	relayMigration, err := os.ReadFile("../../../migrations/133_world_relay_config.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := openLocalRuntimeFixture(t, "CREATE TABLE admin_user(id bigint PRIMARY KEY);INSERT INTO admin_user VALUES(1);"+string(migration)+string(relayMigration))
	repo := NewWorldRepository(db)
	config, err := repo.GetAOIConfig(context.Background())
	if err != nil || config.Enabled || config.CellSizeMilli != 24000 || config.Radius != 1 {
		t.Fatalf("migration default=%+v err=%v", config, err)
	}
	relayEnabled := true
	value, err := repo.UpdateAOIConfig(context.Background(), world.UpdateAOIConfigInput{Enabled: true, RelayEnabled: &relayEnabled, CellSizeMilli: 12000, Radius: 2, Reason: "isolated test", AdminUserID: 1})
	if err != nil || !value.Enabled || !value.RelayEnabled || value.CellSizeMilli != 12000 || value.Radius != 2 || value.AdminUserID != 1 || value.Reason != "isolated test" {
		t.Fatalf("persisted config=%+v err=%v", value, err)
	}
	retained, err := repo.UpdateAOIConfig(context.Background(), world.UpdateAOIConfigInput{Enabled: true, CellSizeMilli: 12000, Radius: 2, Reason: "legacy omitted relay flag", AdminUserID: 1})
	if err != nil || !retained.RelayEnabled {
		t.Fatalf("legacy update cleared relay flag: %+v %v", retained, err)
	}

}
