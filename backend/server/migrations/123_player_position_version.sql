BEGIN;

-- 为玩家永久位置增加单调递增版本。现有玩家从 0 开始，Redis 初始化时会以该值作为后续移动版本基线。
ALTER TABLE player
    ADD COLUMN IF NOT EXISTS position_version BIGINT NOT NULL DEFAULT 0;

-- BIGINT 需要保持非负，避免服务端读取后转换为 uint64 时出现无效的超大版本。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_player_position_version_non_negative'
          AND conrelid = 'player'::regclass
    ) THEN
        ALTER TABLE player
            ADD CONSTRAINT ck_player_position_version_non_negative
            CHECK (position_version >= 0);
    END IF;
END
$$;

COMMENT ON COLUMN player.position_version IS '玩家永久位置的单调递增版本；仅允许更高版本覆盖当前场景和坐标';

COMMIT;

-- 回滚说明：确认所有服务端进程已停止使用位置版本后执行：
-- ALTER TABLE player DROP CONSTRAINT IF EXISTS ck_player_position_version_non_negative;
-- ALTER TABLE player DROP COLUMN IF EXISTS position_version;
