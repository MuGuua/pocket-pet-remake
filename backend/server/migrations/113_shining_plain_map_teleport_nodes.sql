BEGIN;

-- 闪光平原区域地图的快速传送中心由服务端数据库权威维护。
-- 客户端只提交目标 scene_id；中心出生格使用当前服务端已验证的安全出生点。
INSERT INTO world_map_teleport_node (scene_id, center_x, center_y, status)
VALUES
    (9, 14, 8, 1),
    (10, 5, 8, 1),
    (11, 5, 7, 1),
    (12, 5, 7, 1),
    (13, 6, 7, 1),
    (14, 5, 8, 1),
    (15, 5, 9, 1),
    (16, 7, 7, 1),
    (17, 5, 7, 1),
    (18, 5, 7, 1),
    (19, 7, 10, 1),
    (20, 6, 7, 1),
    (21, 9, 7, 1),
    (22, 5, 7, 1),
    (23, 6, 7, 1),
    (24, 5, 7, 1),
    (25, 7, 7, 1)
ON CONFLICT (scene_id) DO UPDATE
SET center_x = EXCLUDED.center_x,
    center_y = EXCLUDED.center_y,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
