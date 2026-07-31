BEGIN;

-- 注册闪光镇东路通往闪光平原的中转地图，以及闪光平原首张地图“闪耀广场”。
-- 门拓扑与出生点由服务端 world repository 维护；客户端只提交 scene_id + portal_id。
INSERT INTO world_scene_definition (scene_id, scene_code, scene_name, status, required_level)
VALUES
    (8, 'shanguang_town_transfer_area', '闪光镇传送区', 1, 1),
    (9, 'shining_square', '闪耀广场', 1, 1)
ON CONFLICT (scene_id) DO UPDATE SET
    scene_code = EXCLUDED.scene_code,
    scene_name = EXCLUDED.scene_name,
    status = EXCLUDED.status,
    required_level = EXCLUDED.required_level,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
