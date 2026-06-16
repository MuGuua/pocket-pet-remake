-- 041_player_vigor_spirit.sql
-- 将原体力字段重命名为活力，并新增战斗技能消耗的精力字段。

ALTER TABLE player RENAME COLUMN energy TO vigor;
ALTER TABLE player RENAME COLUMN energy_max TO vigor_max;

ALTER TABLE player
  ADD COLUMN IF NOT EXISTS spirit INTEGER NOT NULL DEFAULT 40,
  ADD COLUMN IF NOT EXISTS spirit_max INTEGER NOT NULL DEFAULT 40;

-- 存量玩家补齐精力上限，保留迁移前已存在的活力数值。
UPDATE player
SET spirit = 40,
    spirit_max = 40
WHERE spirit = 0 OR spirit_max = 0;

ALTER TABLE player
  ALTER COLUMN vigor SET DEFAULT 100,
  ALTER COLUMN vigor_max SET DEFAULT 100,
  ALTER COLUMN spirit SET DEFAULT 40,
  ALTER COLUMN spirit_max SET DEFAULT 40;
