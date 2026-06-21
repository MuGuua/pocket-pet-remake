-- 054_pet_skill_slots.sql
-- 宠物多分类技能槽：天生/神符/普通/法宝。

ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS innate_skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS normal_skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN pet_definition.innate_skill_ids IS '模板天生技，最多 5 个 skill_id';
COMMENT ON COLUMN pet_definition.normal_skill_ids IS '模板默认普通技，最多 3 个 skill_id';

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS innate_skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS normal_skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS active_talisman_skill_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talisman_hero_skill_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talisman_slot_1_skill_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talisman_slot_2_skill_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talisman_slot_3_skill_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS active_talisman_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS talisman_hero_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS talisman_slot_1_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS talisman_slot_2_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS talisman_slot_3_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN player_pet.innate_skill_ids IS '天生技槽，最多 5 个 skill_id';
COMMENT ON COLUMN player_pet.normal_skill_ids IS '普通技槽，固定 3 个 skill_id';
COMMENT ON COLUMN player_pet.active_talisman_enabled IS '主动神符技是否已由道具开启';

CREATE TABLE IF NOT EXISTS pet_artifact_equipment (
  pet_uid BIGINT NOT NULL,
  player_id BIGINT NOT NULL,
  slot_index SMALLINT NOT NULL,
  skill_id INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (pet_uid, slot_index),
  CONSTRAINT ck_pet_artifact_slot_index CHECK (slot_index >= 0 AND slot_index <= 2),
  CONSTRAINT fk_pet_artifact_equipment_pet FOREIGN KEY (pet_uid) REFERENCES player_pet (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pet_artifact_equipment_player
  ON pet_artifact_equipment (player_id, pet_uid);

CREATE TRIGGER trg_pet_artifact_equipment_updated_at
BEFORE UPDATE ON pet_artifact_equipment
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS pet_skill_slot_unlock_item (
  slot_key VARCHAR(32) PRIMARY KEY,
  item_id INTEGER NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_pet_skill_slot_unlock_item_updated_at
BEFORE UPDATE ON pet_skill_slot_unlock_item
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE pet_skill_slot_unlock_item IS '神符类技能槽解锁道具配置：slot_key=active_talisman|talisman_hero|talisman_1|talisman_2|talisman_3';

-- 存量实例：legacy skill_ids 前 3 个迁入 normal_skill_ids。
UPDATE player_pet pp
SET normal_skill_ids = COALESCE(
  (
    SELECT jsonb_agg(value)
    FROM (
      SELECT elem.value
      FROM jsonb_array_elements_text(pp.skill_ids) WITH ORDINALITY AS elem(value, ord)
      ORDER BY elem.ord
      LIMIT 3
    ) sliced
  ),
  '[]'::jsonb
)
WHERE jsonb_array_length(COALESCE(pp.normal_skill_ids, '[]'::jsonb)) = 0
  AND jsonb_array_length(COALESCE(pp.skill_ids, '[]'::jsonb)) > 0;

-- 模板：同样把 skill_ids 迁入 normal_skill_ids（innate 由运营后续在后台拆分）。
UPDATE pet_definition pd
SET normal_skill_ids = COALESCE(
  (
    SELECT jsonb_agg(value)
    FROM (
      SELECT elem.value
      FROM jsonb_array_elements_text(pd.skill_ids) WITH ORDINALITY AS elem(value, ord)
      ORDER BY elem.ord
      LIMIT 3
    ) sliced
  ),
  '[]'::jsonb
)
WHERE jsonb_array_length(COALESCE(pd.normal_skill_ids, '[]'::jsonb)) = 0
  AND jsonb_array_length(COALESCE(pd.skill_ids, '[]'::jsonb)) > 0;
