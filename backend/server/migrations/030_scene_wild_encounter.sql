-- 030_scene_wild_encounter.sql
-- 地图暗雷遭遇：按 scene_id 配置步进概率与刷怪池，与 monster_encounter（NPC/Boss 固定实体战）分离。

CREATE TABLE IF NOT EXISTS scene_wild_encounter (
  scene_id INTEGER PRIMARY KEY,
  encounter_name VARCHAR(64) NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  -- 每步遭遇概率，万分比：800 表示 8%
  encounter_rate INTEGER NOT NULL DEFAULT 0,
  spawn_monster_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  status INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_scene_wild_encounter_updated_at
BEFORE UPDATE ON scene_wild_encounter
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 北部野外（scene_id=4）启用暗雷，单只野生怪模板 9001
INSERT INTO scene_wild_encounter (scene_id, encounter_name, description, encounter_rate, spawn_monster_ids, status) VALUES
  (4, '北部野外暗雷', '玩家在北部野外移动时由客户端按概率判定，触发后上报服务端开战', 800, '[9001]'::jsonb, 1)
ON CONFLICT (scene_id) DO UPDATE SET
  encounter_name = EXCLUDED.encounter_name,
  description = EXCLUDED.description,
  encounter_rate = EXCLUDED.encounter_rate,
  spawn_monster_ids = EXCLUDED.spawn_monster_ids,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
