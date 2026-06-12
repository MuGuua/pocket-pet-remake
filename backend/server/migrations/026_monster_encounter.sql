-- 026_monster_encounter.sql
-- 世界实体与怪物模板的遭遇配置：按 entity_id 决定刷怪数量与 monster_id 列表。

CREATE TABLE IF NOT EXISTS monster_encounter (
  entity_id BIGINT PRIMARY KEY,
  encounter_name VARCHAR(64) NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  spawn_monster_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  status INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_monster_encounter_updated_at
BEFORE UPDATE ON monster_encounter
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO monster_encounter (entity_id, encounter_name, description, spawn_monster_ids, status) VALUES
  (90001, 'GuideNPC 遭遇', '新手引导战斗', '[9001]'::jsonb, 1),
  (90002, 'StationKeeper 遭遇', '车站守卫战斗', '[9001]'::jsonb, 1),
  (90004, 'NorthFieldScout 遭遇', '北部scout双人战', '[9001, 9001]'::jsonb, 1),
  (90005, 'SchoolCaretaker 遭遇', '学校看护双人战', '[9001, 9001]'::jsonb, 1),
  (90006, 'BattleGuide 遭遇', '战斗教学：攻击+支援', '[9001, 9002]'::jsonb, 1)
ON CONFLICT (entity_id) DO UPDATE SET
  encounter_name = EXCLUDED.encounter_name,
  description = EXCLUDED.description,
  spawn_monster_ids = EXCLUDED.spawn_monster_ids,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
