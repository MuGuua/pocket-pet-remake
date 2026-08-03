BEGIN;

-- 注册闪光平原当前已落地的地图资源。普通传送门拓扑与出生格由服务端
-- world repository 统一判定，客户端只提交 scene_id 与 portal_id。
INSERT INTO world_scene_definition (scene_id, scene_code, scene_name, status, required_level)
VALUES
    (10, 'shining_plain_pet_school', '闪光平原宠物学校', 1, 1),
    (11, 'ice_dream', '冰雪梦境', 1, 1),
    (12, 'ash_dream', '灰烬梦境', 1, 1),
    (13, 'emerald_dream', '翡翠梦境', 1, 1),
    (14, 'aer_room', '阿尔的房间', 1, 1),
    (15, 'shining_plain_office', '办公区', 1, 1),
    (16, 'shining_plain_commercial', '商业区', 1, 1),
    (17, 'shining_plain_registration', '报名区', 1, 1),
    (18, 'shining_plain_preparation', '准备区', 1, 1),
    (19, 'family_hall', '家族会馆', 1, 1),
    (20, 'shining_south_road', '闪光南路', 1, 1),
    (21, 'colorful_lake', '五彩湖', 1, 1),
    (22, 'swamp', '沼泽地', 1, 1),
    (23, 'shining_coast', '闪光海岸', 1, 1),
    (24, 'mud_land', '尘泥之地', 1, 1),
    (25, 'spirit_hall', '精灵大厅', 1, 1)
ON CONFLICT (scene_id) DO UPDATE SET
    scene_code = EXCLUDED.scene_code,
    scene_name = EXCLUDED.scene_name,
    status = EXCLUDED.status,
    required_level = EXCLUDED.required_level,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
