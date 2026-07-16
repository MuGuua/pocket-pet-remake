-- 注册镇北兔子 NPC；坐标与朝向由客户端地图场景资源维护。
INSERT INTO world_entity_definition (
  entity_id,
  entity_code,
  display_name,
  entity_type,
  scene_id,
  status
) VALUES (
  92002,
  'rabbit',
  '兔子',
  2,
  4,
  1
)
ON CONFLICT (entity_id) DO UPDATE SET
  entity_code = EXCLUDED.entity_code,
  display_name = EXCLUDED.display_name,
  entity_type = EXCLUDED.entity_type,
  scene_id = EXCLUDED.scene_id,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
