package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// 真库执行正式迁移，缺少的宠物源表仅用于隔离结构；实际人物视图沿用完整源/公式测试夹具。
func versionFixture(t *testing.T) *PlayerRepository {
	t.Helper()
	db := openLocalRuntimeFixture(t, playerDisplayFixtureSQL()+`
 CREATE TABLE player_pet(id bigint,player_id bigint);
 CREATE TABLE player_lineup(player_id bigint);
 CREATE TABLE pet_artifact_equipment(player_id bigint);
 CREATE TABLE player_skill_progress(player_id bigint);
 CREATE TABLE pet_definition(pet_id bigint);
 CREATE TABLE pet_combat_stat_cap(id bigint);`)
	migration, err := os.ReadFile("../../../migrations/132_runtime_snapshot_versions.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeSnapshotSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return NewPlayerRepository(db, true)
}

// 不变源只读取版本，个人写入、装备变化、配置修改与回滚都与真实人物快照一致。
func TestSnapshotVersionInvalidationPostgres(t *testing.T) {
	repo := versionFixture(t)
	ctx := context.Background()
	for _, mutation := range []string{"", `UPDATE player SET strength=9 WHERE id=1`, `UPDATE equipment_instance SET enhance_level=2`, `UPDATE player_attr_convert_config SET convert_rate=4`} {
		if mutation != "" {
			if _, err := repo.db.ExecContext(ctx, mutation); err != nil {
				t.Fatal(err)
			}
		}
		if err := repo.RefreshPlayerCombatSnapshot(ctx, 1); err != nil {
			t.Fatal(err)
		}
		expected, err := repo.FindDisplayProfile(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := repo.FindPlayerCombatSnapshot(ctx, 1)
		if err != nil || actual.ATK != expected.ATK {
			t.Fatalf("stale view: %+v %+v %v", actual, expected, err)
		}
		calls := 0
		if err := refreshVersionedSnapshot(ctx, repo.db, 1, "player", func(DBTX) error { calls++; return nil }); err != nil || calls != 0 {
			t.Fatalf("clean view rebuilt %d %v", calls, err)
		}
	}
	var before, after int64
	_ = repo.db.QueryRowContext(ctx, `SELECT version FROM runtime_snapshot_source WHERE player_id=1`).Scan(&before)
	_, err := repo.db.ExecContext(ctx, `UPDATE player SET pos_x=3,position_version=position_version+1 WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.db.QueryRowContext(ctx, `SELECT version FROM runtime_snapshot_source WHERE player_id=1`).Scan(&after)
	if before != after {
		t.Fatal("movement invalidated combat view")
	}
	tx, err := repo.db.(txBeginner).BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE player SET strength=100 WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	_ = repo.db.QueryRowContext(ctx, `SELECT version FROM runtime_snapshot_source WHERE player_id=1`).Scan(&after)
	if before != after {
		t.Fatal("rollback published invalidation")
	}
}

// 并发读只发布一次；计算失败时视图修改和新鲜标记共同回滚，随后可重试恢复。
func TestSnapshotVersionConcurrentAndFailurePostgres(t *testing.T) {
	repo := versionFixture(t)
	ctx := context.Background()
	if err := repo.RefreshPlayerCombatSnapshot(ctx, 1); err != nil {
		t.Fatal(err)
	}
	original, err := repo.FindPlayerCombatSnapshot(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callErr := refreshVersionedSnapshot(ctx, repo.db, 1, "skills", func(db DBTX) error { calls.Add(1); _, err := db.ExecContext(ctx, `SELECT pg_sleep(0.01)`); return err })
			if callErr != nil {
				t.Error(callErr)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("duplicate rebuilds: %d", calls.Load())
	}
	sentinel := errors.New("failed rebuild")
	if _, err := repo.db.ExecContext(ctx, `UPDATE player SET strength=6 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	err = refreshVersionedSnapshot(ctx, repo.db, 1, "skills", func(db DBTX) error {
		_, err := db.ExecContext(ctx, `UPDATE player_combat_snapshot SET atk=999 WHERE player_id=1`)
		if err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	remaining, err := repo.FindPlayerCombatSnapshot(ctx, 1)
	if err != nil || remaining.ATK != original.ATK {
		t.Fatal("failed rebuild leaked partial view", err)
	}
	var fresh bool
	if err := repo.db.QueryRowContext(ctx, snapshotFreshQuery, 1, "skills").Scan(&fresh); err != nil || fresh {
		t.Fatalf("failure published freshness %v %v", fresh, err)
	}
	if err := refreshVersionedSnapshot(ctx, repo.db, 1, "skills", func(DBTX) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

// 计算中源数据提交会使第一次结果回滚，第二次必须使用新版本，不能发布混合源结果。
func TestSnapshotVersionMutationDuringBuildPostgres(t *testing.T) {
	repo := versionFixture(t)
	ctx := context.Background()
	calls := 0
	err := refreshVersionedSnapshot(ctx, repo.db, 1, "player", func(db DBTX) error {
		calls++
		scoped := NewPlayerRepository(db)
		if err := scoped.rebuildPlayerCombatSnapshot(ctx, 1); err != nil {
			return err
		}
		if calls == 1 {
			_, err := repo.db.ExecContext(ctx, `UPDATE player SET strength=20 WHERE id=1`)
			return err
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("rebuild failed to retry source mutation: calls=%d err=%v", calls, err)
	}
	actual, err := repo.FindPlayerCombatSnapshot(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := repo.FindDisplayProfile(ctx, 1)
	if err != nil || actual.ATK != expected.ATK {
		t.Fatal("mixed source snapshot published", err)
	}
}
