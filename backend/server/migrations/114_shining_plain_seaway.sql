BEGIN;

-- 注册用户已落地的海道场景。普通传送门拓扑继续由服务端 world repository
-- 校验，客户端只提交目标场景、来源门编号和目标地图入口格。
INSERT INTO world_scene_definition (
    scene_id,
    scene_code,
    scene_name,
    status,
    required_level
)
VALUES
    (26, 'seaway', '海道', 1, 1)
ON CONFLICT (scene_id) DO UPDATE SET
    scene_code = EXCLUDED.scene_code,
    scene_name = EXCLUDED.scene_name,
    status = EXCLUDED.status,
    required_level = EXCLUDED.required_level,
    updated_at = CURRENT_TIMESTAMP;

-- 开放海道世界地图快速传送；中心格使用海道主通路中的无阻挡安全位置。
INSERT INTO world_map_teleport_node (
    scene_id,
    center_x,
    center_y,
    status
)
VALUES
    (26, 6, 7, 1)
ON CONFLICT (scene_id) DO UPDATE SET
    center_x = EXCLUDED.center_x,
    center_y = EXCLUDED.center_y,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
