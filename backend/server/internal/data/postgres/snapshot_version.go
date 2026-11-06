package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// 两个源版本分别覆盖个人数据和公共配置；版本在源写入事务中推进，不依赖进程内缓存。
const snapshotFreshQuery = `SELECT EXISTS (
 SELECT 1 FROM runtime_snapshot_stamp s
 JOIN runtime_snapshot_source p ON p.player_id=s.player_id
 JOIN runtime_snapshot_source g ON g.player_id=0
 WHERE s.player_id=$1 AND s.view_name=$2 AND s.player_version=p.version AND s.config_version=g.version
)`

// 清洁读只做一次版本判定；脏读串行化同一玩家的重建后重查，避免并发读重复重建。
// 源数据触发器在同一事务推进版本，因此失败/回滚不会错误发布新鲜标记。
func refreshVersionedSnapshot(ctx context.Context, db DBTX, playerID uint64, view string, rebuild func(DBTX) error) error {
	if playerID == 0 {
		return rebuild(db)
	}
	beginner, ok := db.(txBeginner)
	if !ok {
		return rebuild(db)
	}
	var fresh bool
	if err := db.QueryRowContext(ctx, snapshotFreshQuery, playerID, view).Scan(&fresh); err != nil {
		return fmt.Errorf("read snapshot source version (apply migration 132): %w", err)
	}
	if fresh {
		return nil
	}

	// 仅串行化同一玩家的重建，不锁源版本行；源写入可能同时改个人/公共配置，不能反向抢锁。
	for attempt := 0; attempt < 3; attempt++ {
		changed, err := rebuildSnapshotVersion(ctx, beginner, playerID, view, rebuild)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
	}
	return fmt.Errorf("snapshot sources changed repeatedly during rebuild")
}

// 重建前后版本一致才发布；任一源在计算中变化，整份派生视图回滚并重试，不能标成新鲜。
func rebuildSnapshotVersion(ctx context.Context, beginner txBeginner, playerID uint64, view string, rebuild func(DBTX) error) (bool, error) {
	tx, err := beginner.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, err
	}
	defer rollbackTx(tx)
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('runtime_snapshot:' || $1::bigint::text,0))`, playerID); err != nil {
		return false, err
	}
	var fresh bool
	if err := tx.QueryRowContext(ctx, snapshotFreshQuery, playerID, view).Scan(&fresh); err != nil {
		return false, err
	}
	if fresh {
		return false, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_snapshot_source(player_id) VALUES($1) ON CONFLICT DO NOTHING`, playerID); err != nil {
		return false, err
	}
	const versions = `SELECT p.version,g.version FROM runtime_snapshot_source p CROSS JOIN runtime_snapshot_source g WHERE p.player_id=$1 AND g.player_id=0`
	var playerVersion, configVersion, latestPlayer, latestConfig int64
	if err := tx.QueryRowContext(ctx, versions, playerID).Scan(&playerVersion, &configVersion); err != nil {
		return false, err
	}
	if err := rebuild(tx); err != nil {
		return false, err
	}
	if err := tx.QueryRowContext(ctx, versions, playerID).Scan(&latestPlayer, &latestConfig); err != nil {
		return false, err
	}
	if latestPlayer != playerVersion || latestConfig != configVersion {
		return true, nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_snapshot_stamp(player_id,view_name,player_version,config_version)
 VALUES($1,$2,$3,$4) ON CONFLICT(player_id,view_name) DO UPDATE SET
 player_version=EXCLUDED.player_version,config_version=EXCLUDED.config_version,updated_at=CURRENT_TIMESTAMP`, playerID, view, playerVersion, configVersion); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

// 正式启动必须确认版本表、公共版本及全部源触发器已就绪，避免漏迁移后把旧视图当作新鲜。
func ValidateRuntimeSnapshotSchema(ctx context.Context, db DBTX) error {
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM runtime_snapshot_source WHERE player_id=0)
 AND to_regclass('runtime_snapshot_stamp') IS NOT NULL
 AND (SELECT count(*) FROM pg_trigger WHERE tgname='runtime_snapshot_invalidate' AND tgenabled IN ('O','A')
 AND tgrelid IN ('player'::regclass,'player_pet'::regclass,'player_lineup'::regclass,'pet_artifact_equipment'::regclass,
 'player_equipment_slot'::regclass,'player_skill_progress'::regclass,'equipment_instance'::regclass,'item_definition'::regclass,
 'item_equipment_extra'::regclass,'player_attr_convert_config'::regclass,'pet_definition'::regclass,'pet_combat_stat_cap'::regclass))=12`).Scan(&ready)
	if err != nil {
		return fmt.Errorf("validate snapshot schema (apply migration 132): %w", err)
	}
	if !ready {
		return fmt.Errorf("snapshot source triggers unavailable: apply migration 132 before deployment")
	}
	return nil
}
