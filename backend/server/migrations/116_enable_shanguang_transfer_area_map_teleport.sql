BEGIN;

-- 开放闪光镇地图“通往闪光平原”标点到闪光镇传送区的快速旅行。
-- center_x / center_y 只用于服务端现有快速传送校验与位置持久化；
-- 客户端实际显示落点仍从目标地图脚本的 login_and_map_teleport_spawn_position 读取。
INSERT INTO world_map_teleport_node (
    scene_id,
    center_x,
    center_y,
    status
)
VALUES
    (8, 5, 10, 1)
ON CONFLICT (scene_id) DO UPDATE SET
    center_x = EXCLUDED.center_x,
    center_y = EXCLUDED.center_y,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
