-- NPC 坐标改由客户端场景资源摆放；服务端 world_entity_definition 仅保留所属 scene_id。
-- 同时引入 world_scene_definition，供后台展示场景中文名。

CREATE TABLE IF NOT EXISTS world_scene_definition (
  scene_id INTEGER PRIMARY KEY,
  scene_code TEXT NOT NULL UNIQUE,
  scene_name TEXT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO world_scene_definition (scene_id, scene_code, scene_name, status) VALUES
  (1, 'roxus_house', '洛克斯小屋', 1),
  (2, 'east_road_of_shanguang_town', '闪光镇东路', 1),
  (3, 'radiant_market', '闪光市场', 1),
  (4, 'bei_lu', '北路', 1),
  (5, 'xue_xiao', '学校', 1),
  (6, 'da_guai_qu', '打怪区', 1)
ON CONFLICT (scene_id) DO UPDATE SET
  scene_code = EXCLUDED.scene_code,
  scene_name = EXCLUDED.scene_name,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

ALTER TABLE world_entity_definition
  DROP COLUMN IF EXISTS pos_x,
  DROP COLUMN IF EXISTS pos_y,
  DROP COLUMN IF EXISTS dir,
  DROP COLUMN IF EXISTS speed;
