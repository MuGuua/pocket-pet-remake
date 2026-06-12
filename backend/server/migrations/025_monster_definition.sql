-- 025_monster_definition.sql
-- 系统怪物模板：战斗中的怪物单位统一引用 monster_id 与 skill_id。

CREATE TABLE IF NOT EXISTS monster_definition (
  monster_id INTEGER PRIMARY KEY,
  monster_name VARCHAR(64) NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  level INTEGER NOT NULL DEFAULT 1,
  quality INTEGER NOT NULL DEFAULT 1,
  hp INTEGER NOT NULL DEFAULT 1,
  hp_max INTEGER NOT NULL DEFAULT 1,
  atk INTEGER NOT NULL DEFAULT 1,
  def INTEGER NOT NULL DEFAULT 1,
  spd INTEGER NOT NULL DEFAULT 1,
  mana INTEGER NOT NULL DEFAULT 0,
  skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  status INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_monster_definition_updated_at
BEFORE UPDATE ON monster_definition
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO monster_definition (
  monster_id, monster_name, description, level, quality, hp, hp_max, atk, def, spd, mana, skill_ids, status
) VALUES
  (9001, '野生怪物', '默认野外战斗怪物模板', 1, 1, 22, 22, 12, 9, 8, 9, '[90001, 90002]'::jsonb, 1),
  (9002, '野性支援', '带治疗技能的怪物模板', 1, 1, 20, 20, 11, 10, 9, 12, '[90002, 90003]'::jsonb, 1)
ON CONFLICT (monster_id) DO UPDATE SET
  monster_name = EXCLUDED.monster_name,
  description = EXCLUDED.description,
  level = EXCLUDED.level,
  quality = EXCLUDED.quality,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  mana = EXCLUDED.mana,
  skill_ids = EXCLUDED.skill_ids,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
