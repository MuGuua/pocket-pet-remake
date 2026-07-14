-- 注册一次性剧情场景“时光小屋”。该场景仅允许通过 portal_id=7001 单向离开到闪光镇东路。
-- 东路不配置返回时光小屋的传送门，玩家离开后不能从普通地图重新进入。

INSERT INTO world_scene_definition (scene_id, scene_code, scene_name, status)
VALUES (7, 'time_house', '时光小屋', 1)
ON CONFLICT (scene_id) DO UPDATE SET
  scene_code = EXCLUDED.scene_code,
  scene_name = EXCLUDED.scene_name,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
