-- 035_player_level_progression.sql
-- 玩家等级经验、自由属性点与基础属性加点体系。
-- level 表示玩家当前等级；exp 表示当前等级已累计、尚未升级的经验。
-- player_level_config.level 表示「处于该等级时升到下一级」所需经验与升级奖励。

CREATE TABLE IF NOT EXISTS player_level_config (
  level INTEGER PRIMARY KEY,
  exp_required BIGINT NOT NULL DEFAULT 0,
  attr_points INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_player_level_config_level CHECK (level >= 1 AND level <= 100),
  CONSTRAINT ck_player_level_config_exp_required CHECK (exp_required >= 0),
  CONSTRAINT ck_player_level_config_attr_points CHECK (attr_points >= 0)
);

CREATE TRIGGER trg_player_level_config_updated_at
BEFORE UPDATE ON player_level_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS player_attr_convert_config (
  id BIGSERIAL PRIMARY KEY,
  source_attr VARCHAR(16) NOT NULL,
  target_attr VARCHAR(32) NOT NULL,
  convert_rate INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_attr_convert UNIQUE (source_attr, target_attr),
  CONSTRAINT ck_player_attr_convert_rate CHECK (convert_rate >= 0)
);

CREATE TRIGGER trg_player_attr_convert_config_updated_at
BEFORE UPDATE ON player_attr_convert_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE player
  ADD COLUMN IF NOT EXISTS free_attr_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS strength INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS vitality INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS agility INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS mind INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS base_hp_max INTEGER NOT NULL DEFAULT 100,
  ADD COLUMN IF NOT EXISTS base_atk INTEGER NOT NULL DEFAULT 24,
  ADD COLUMN IF NOT EXISTS base_def INTEGER NOT NULL DEFAULT 12,
  ADD COLUMN IF NOT EXISTS base_spd INTEGER NOT NULL DEFAULT 18,
  ADD COLUMN IF NOT EXISTS base_mana INTEGER NOT NULL DEFAULT 20,
  ADD COLUMN IF NOT EXISTS base_hit_pct INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS base_dodge_pct INTEGER NOT NULL DEFAULT 6;

-- 将现有战斗属性复制为裸装基础值，保证迁移后未加点玩家战力不变。
UPDATE player
SET base_hp_max = hp_max,
    base_atk = atk,
    base_def = def,
    base_spd = spd,
    base_mana = mana,
    base_hit_pct = hit_pct,
    base_dodge_pct = dodge_pct
WHERE base_hp_max = 100
  AND base_atk = 24
  AND base_def = 12;

CREATE TABLE IF NOT EXISTS player_attr_allocate_log (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  delta_strength INTEGER NOT NULL DEFAULT 0,
  delta_vitality INTEGER NOT NULL DEFAULT 0,
  delta_agility INTEGER NOT NULL DEFAULT 0,
  delta_mind INTEGER NOT NULL DEFAULT 0,
  free_before INTEGER NOT NULL DEFAULT 0,
  free_after INTEGER NOT NULL DEFAULT 0,
  reason_type VARCHAR(32) NOT NULL DEFAULT 'manual_allocate',
  operator_type VARCHAR(32) NOT NULL DEFAULT 'player',
  operator_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_player_attr_allocate_log_player_id
  ON player_attr_allocate_log (player_id, created_at DESC);

-- 1~100 级经验与升级属性点种子。公式可在后台继续微调。
INSERT INTO player_level_config (level, exp_required, attr_points, status)
SELECT
  gs.level,
  CASE
    WHEN gs.level >= 100 THEN 0
    ELSE GREATEST(100, (100 * power(gs.level::numeric, 1.8))::bigint)
  END AS exp_required,
  CASE
    WHEN gs.level >= 100 THEN 0
    ELSE 1
  END AS attr_points,
  1 AS status
FROM generate_series(1, 100) AS gs(level)
ON CONFLICT (level) DO UPDATE SET
  exp_required = EXCLUDED.exp_required,
  attr_points = EXCLUDED.attr_points,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO player_attr_convert_config (source_attr, target_attr, convert_rate, status)
VALUES
  ('strength', 'atk', 3, 1),
  ('vitality', 'hp_max', 50, 1),
  ('vitality', 'def', 2, 1),
  ('agility', 'spd', 2, 1),
  ('agility', 'dodge_pct', 1, 1),
  ('mind', 'mana', 4, 1)
ON CONFLICT (source_attr, target_attr) DO UPDATE SET
  convert_rate = EXCLUDED.convert_rate,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
