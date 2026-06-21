-- 058_player_equipment_foundation.sql
-- 玩家人物装备系统底座：扩展装备模板、佩戴槽、强化成功率、镶嵌、套装与药囊配置。

ALTER TABLE item_equipment_extra
  ADD COLUMN IF NOT EXISTS can_enhance BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS max_enhance_level INTEGER NOT NULL DEFAULT 15,
  ADD COLUMN IF NOT EXISTS set_id BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS appearance_skin_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS appearance_only BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS base_stats_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS enhance_per_level_stats_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS socket_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS allowed_gem_types_json JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_extra_max_enhance
    CHECK (max_enhance_level >= 0 AND max_enhance_level <= 15);
ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_extra_socket_count
    CHECK (socket_count >= 0 AND socket_count <= 8);

COMMENT ON COLUMN item_equipment_extra.base_stats_json IS '次要战斗属性 JSON，键与 pet_combat_stat_cap 对齐';
COMMENT ON COLUMN item_equipment_extra.enhance_per_level_stats_json IS '每强化 1 级线性增加的属性 JSON';
COMMENT ON COLUMN item_equipment_extra.appearance_only IS '时装等仅外观装备为 true，不参与战斗属性聚合';

CREATE TABLE IF NOT EXISTS player_equipment_slot (
  player_id BIGINT NOT NULL REFERENCES player(id),
  equip_slot VARCHAR(32) NOT NULL,
  item_uid VARCHAR(64) NOT NULL REFERENCES equipment_instance(item_uid),
  equipped_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, equip_slot),
  UNIQUE (item_uid)
);

CREATE INDEX IF NOT EXISTS idx_player_equipment_slot_player ON player_equipment_slot (player_id);

CREATE TABLE IF NOT EXISTS equipment_enhance_success_config (
  target_level INTEGER PRIMARY KEY,
  success_rate_pct INTEGER NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_enhance_success_rate CHECK (success_rate_pct >= 0 AND success_rate_pct <= 100),
  CONSTRAINT chk_equipment_enhance_target_level CHECK (target_level >= 1 AND target_level <= 15)
);

DROP TRIGGER IF EXISTS trg_equipment_enhance_success_config_updated_at ON equipment_enhance_success_config;
CREATE TRIGGER trg_equipment_enhance_success_config_updated_at
BEFORE UPDATE ON equipment_enhance_success_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO equipment_enhance_success_config (target_level, success_rate_pct, description, status)
VALUES
  (1, 100, '强化至 +1', 1),
  (2, 100, '强化至 +2', 1),
  (3, 100, '强化至 +3', 1),
  (4, 90, '强化至 +4', 1),
  (5, 90, '强化至 +5', 1),
  (6, 90, '强化至 +6', 1),
  (7, 75, '强化至 +7', 1),
  (8, 75, '强化至 +8', 1),
  (9, 65, '强化至 +9', 1),
  (10, 55, '强化至 +10', 1),
  (11, 45, '强化至 +11', 1),
  (12, 35, '强化至 +12', 1),
  (13, 25, '强化至 +13', 1),
  (14, 15, '强化至 +14', 1),
  (15, 10, '强化至 +15', 1)
ON CONFLICT (target_level) DO UPDATE SET
  success_rate_pct = EXCLUDED.success_rate_pct,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS equipment_instance_socket (
  item_uid VARCHAR(64) NOT NULL REFERENCES equipment_instance(item_uid) ON DELETE CASCADE,
  socket_index INTEGER NOT NULL,
  gem_item_id BIGINT NULL REFERENCES item_definition(item_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (item_uid, socket_index),
  CONSTRAINT chk_equipment_instance_socket_index CHECK (socket_index >= 0)
);

DROP TRIGGER IF EXISTS trg_equipment_instance_socket_updated_at ON equipment_instance_socket;
CREATE TRIGGER trg_equipment_instance_socket_updated_at
BEFORE UPDATE ON equipment_instance_socket
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS equipment_set_definition (
  set_id BIGSERIAL PRIMARY KEY,
  set_code VARCHAR(64) NOT NULL UNIQUE,
  set_name VARCHAR(128) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  max_pieces INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_set_max_pieces CHECK (max_pieces >= 0)
);

DROP TRIGGER IF EXISTS trg_equipment_set_definition_updated_at ON equipment_set_definition;
CREATE TRIGGER trg_equipment_set_definition_updated_at
BEFORE UPDATE ON equipment_set_definition
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS equipment_set_piece (
  set_id BIGINT NOT NULL REFERENCES equipment_set_definition(set_id) ON DELETE CASCADE,
  item_id BIGINT NOT NULL REFERENCES item_definition(item_id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (set_id, item_id)
);

CREATE TABLE IF NOT EXISTS equipment_set_effect (
  id BIGSERIAL PRIMARY KEY,
  set_id BIGINT NOT NULL REFERENCES equipment_set_definition(set_id) ON DELETE CASCADE,
  piece_count INTEGER NOT NULL,
  effect_name VARCHAR(128) NOT NULL DEFAULT '',
  effect_desc TEXT NOT NULL DEFAULT '',
  stats_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  special_effect_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_set_effect_piece_count CHECK (piece_count >= 1),
  UNIQUE (set_id, piece_count)
);

DROP TRIGGER IF EXISTS trg_equipment_set_effect_updated_at ON equipment_set_effect;
CREATE TRIGGER trg_equipment_set_effect_updated_at
BEFORE UPDATE ON equipment_set_effect
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS item_medicine_pouch_extra (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id) ON DELETE CASCADE,
  restore_player_hp BOOLEAN NOT NULL DEFAULT TRUE,
  restore_player_spirit BOOLEAN NOT NULL DEFAULT TRUE,
  restore_player_vigor BOOLEAN NOT NULL DEFAULT TRUE,
  restore_pet_hp BOOLEAN NOT NULL DEFAULT TRUE,
  restore_pet_spirit BOOLEAN NOT NULL DEFAULT TRUE,
  restore_lineup_pets BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS trg_item_medicine_pouch_extra_updated_at ON item_medicine_pouch_extra;
CREATE TRIGGER trg_item_medicine_pouch_extra_updated_at
BEFORE UPDATE ON item_medicine_pouch_extra
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS item_gem_extra (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id) ON DELETE CASCADE,
  gem_type VARCHAR(32) NOT NULL DEFAULT '',
  stats_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS trg_item_gem_extra_updated_at ON item_gem_extra;
CREATE TRIGGER trg_item_gem_extra_updated_at
BEFORE UPDATE ON item_gem_extra
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
