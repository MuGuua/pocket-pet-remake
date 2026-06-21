-- 051_pet_progression.sql
-- 宠物等级经验、自由属性点、资质拆分与资质→战斗属性转化率配置。

CREATE TABLE IF NOT EXISTS pet_level_config (
  level INTEGER PRIMARY KEY,
  exp_required BIGINT NOT NULL DEFAULT 0,
  attr_points INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_pet_level_config_level CHECK (level >= 1 AND level <= 100),
  CONSTRAINT ck_pet_level_config_exp_required CHECK (exp_required >= 0),
  CONSTRAINT ck_pet_level_config_attr_points CHECK (attr_points >= 0)
);

CREATE TRIGGER trg_pet_level_config_updated_at
BEFORE UPDATE ON pet_level_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS pet_attr_convert_config (
  attr_type VARCHAR(16) PRIMARY KEY,
  convert_rate NUMERIC(12, 4) NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_pet_attr_convert_rate CHECK (convert_rate > 0)
);

CREATE TRIGGER trg_pet_attr_convert_config_updated_at
BEFORE UPDATE ON pet_attr_convert_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS aptitude_profile VARCHAR(16) NOT NULL DEFAULT 'normal';

COMMENT ON COLUMN pet_definition.aptitude_profile IS '资质倍率档：normal/special/arctic，对应宠物有效资质公式倍率';

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS base_hp_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS base_atk_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS base_def_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS base_spd_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS base_mana_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS extra_hp_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS extra_atk_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS extra_def_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS extra_spd_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS extra_mana_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS free_attr_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS alloc_hp_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS alloc_atk_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS alloc_spd_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS alloc_mana_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS alloc_def_points INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS evolution_level INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS rebirth_level INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS pet_attr_allocate_log (
  id BIGSERIAL PRIMARY KEY,
  pet_uid BIGINT NOT NULL,
  player_id BIGINT NOT NULL,
  delta_hp INTEGER NOT NULL DEFAULT 0,
  delta_atk INTEGER NOT NULL DEFAULT 0,
  delta_spd INTEGER NOT NULL DEFAULT 0,
  delta_mana INTEGER NOT NULL DEFAULT 0,
  delta_def INTEGER NOT NULL DEFAULT 0,
  free_before INTEGER NOT NULL DEFAULT 0,
  free_after INTEGER NOT NULL DEFAULT 0,
  reason_type VARCHAR(32) NOT NULL DEFAULT 'manual_allocate',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pet_attr_allocate_log_pet_uid
  ON pet_attr_allocate_log (pet_uid, created_at DESC);

-- 1~100 级经验与升级属性点种子（与玩家曲线一致，可在后台微调）。
INSERT INTO pet_level_config (level, exp_required, attr_points, status)
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

INSERT INTO pet_attr_convert_config (attr_type, convert_rate, status)
VALUES
  ('hp_max', 27.77, 1),
  ('atk', 277.77, 1),
  ('spd', 2081.51, 1),
  ('mana', 1388.73, 1),
  ('def', 277.77, 1)
ON CONFLICT (attr_type) DO UPDATE SET
  convert_rate = EXCLUDED.convert_rate,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- 回填基础/红色资质：模板值为 base，实例与模板差值为 extra。
UPDATE player_pet pp
SET
  base_hp_apt = pd.hp_apt,
  base_atk_apt = pd.atk_apt,
  base_def_apt = pd.def_apt,
  base_spd_apt = pd.spd_apt,
  base_mana_apt = pd.mana_apt,
  extra_hp_apt = GREATEST(0, pp.hp_apt - pd.hp_apt),
  extra_atk_apt = GREATEST(0, pp.atk_apt - pd.atk_apt),
  extra_def_apt = GREATEST(0, pp.def_apt - pd.def_apt),
  extra_spd_apt = GREATEST(0, pp.spd_apt - pd.spd_apt),
  extra_mana_apt = GREATEST(0, pp.mana_apt - pd.mana_apt)
FROM pet_definition pd
WHERE pp.pet_id = pd.pet_id;
