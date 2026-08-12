BEGIN;

-- 场景矩形边界使用千分之一场景格定点整数，作为普通移动的服务端权威外边界。
-- 初值来自当前 Godot 正式地图的边框或主 TileMapLayer 使用范围，并额外覆盖现有服务端传送出生点。
ALTER TABLE world_scene_definition
    ADD COLUMN IF NOT EXISTS boundary_min_x_milli INTEGER,
    ADD COLUMN IF NOT EXISTS boundary_min_y_milli INTEGER,
    ADD COLUMN IF NOT EXISTS boundary_max_x_milli INTEGER,
    ADD COLUMN IF NOT EXISTS boundary_max_y_milli INTEGER,
    ADD COLUMN IF NOT EXISTS boundary_last_update_reason VARCHAR(500),
    ADD COLUMN IF NOT EXISTS boundary_updated_by_admin_user_id BIGINT REFERENCES admin_user(id);

-- P0-05 只提供矩形外边界；墙体、装饰阻挡和精细通行区域由后续 P0-06 静态通行数据负责。
UPDATE world_scene_definition AS scene
SET boundary_min_x_milli = initial.min_x_milli,
    boundary_min_y_milli = initial.min_y_milli,
    boundary_max_x_milli = initial.max_x_milli,
    boundary_max_y_milli = initial.max_y_milli,
    boundary_last_update_reason = 'P0-05 根据正式地图资源初始化场景矩形边界',
    boundary_updated_by_admin_user_id = NULL,
    updated_at = CURRENT_TIMESTAMP
FROM (VALUES
    (1,      0,     0, 10000, 14000),
    (2,      0, -4000, 10000,  8000),
    (3,      0,     0, 13000, 13000),
    (4,      0,     0,  8000, 10000),
    (5,      0,     0, 12000, 12000),
    (6,      0,     0,  9000, 11000),
    (7,      0,     0,  9000, 11000),
    (8,   1000,  1000, 11000, 14000),
    (9,   1000,  1000, 24000, 14000),
    (10,     0,     0, 10000, 12000),
    (11,     0,     0, 10000, 12000),
    (12,     0,     0, 10000, 12000),
    (13,     0,     0, 12000, 12000),
    (14,     0,     0, 10000, 12000),
    (15,     0,     0, 10000, 17000),
    (16,     0,     0, 14000, 12000),
    (17,     0,     0, 10000, 12000),
    (18,     0,     0, 10000, 12000),
    (19,     0,     0, 14000, 19000),
    (20,     0,     0, 12000, 13000),
    (21,     0,     0, 17000, 12000),
    (22,     0,     0, 10000, 12000),
    (23,     0,     0, 12000, 13000),
    (24,     0,     0, 10000, 12000),
    (25,     0,     0, 14000, 12000),
    (26,     0,     0, 11000, 13000)
) AS initial(scene_id, min_x_milli, min_y_milli, max_x_milli, max_y_milli)
WHERE scene.scene_id = initial.scene_id
  AND (
      scene.boundary_min_x_milli IS NULL
      OR scene.boundary_min_y_milli IS NULL
      OR scene.boundary_max_x_milli IS NULL
      OR scene.boundary_max_y_milli IS NULL
      OR scene.boundary_last_update_reason IS NULL
  );

-- 若数据库存在尚未纳入本次地图资源清单的场景，迁移应明确失败，避免服务端带着未知边界启动。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM world_scene_definition
        WHERE boundary_min_x_milli IS NULL
           OR boundary_min_y_milli IS NULL
           OR boundary_max_x_milli IS NULL
           OR boundary_max_y_milli IS NULL
           OR boundary_last_update_reason IS NULL
           OR BTRIM(boundary_last_update_reason) = ''
    ) THEN
        RAISE EXCEPTION 'world_scene_definition contains scenes without initialized movement boundaries';
    END IF;
END
$$;

ALTER TABLE world_scene_definition
    ALTER COLUMN boundary_min_x_milli SET NOT NULL,
    ALTER COLUMN boundary_min_y_milli SET NOT NULL,
    ALTER COLUMN boundary_max_x_milli SET NOT NULL,
    ALTER COLUMN boundary_max_y_milli SET NOT NULL,
    ALTER COLUMN boundary_last_update_reason SET NOT NULL;

ALTER TABLE world_scene_definition
    DROP CONSTRAINT IF EXISTS ck_world_scene_definition_boundary_order,
    DROP CONSTRAINT IF EXISTS ck_world_scene_definition_boundary_range,
    DROP CONSTRAINT IF EXISTS ck_world_scene_definition_boundary_reason;

ALTER TABLE world_scene_definition
    ADD CONSTRAINT ck_world_scene_definition_boundary_order CHECK (
        boundary_max_x_milli > boundary_min_x_milli
        AND boundary_max_y_milli > boundary_min_y_milli
    ),
    ADD CONSTRAINT ck_world_scene_definition_boundary_range CHECK (
        boundary_min_x_milli >= -10000000
        AND boundary_min_y_milli >= -10000000
        AND boundary_max_x_milli <= 10000000
        AND boundary_max_y_milli <= 10000000
    ),
    ADD CONSTRAINT ck_world_scene_definition_boundary_reason CHECK (
        CHAR_LENGTH(BTRIM(boundary_last_update_reason)) BETWEEN 1 AND 500
    );

COMMENT ON COLUMN world_scene_definition.boundary_min_x_milli IS '人物中心可到达矩形的最小 X，单位为千分之一场景格，包含端点';
COMMENT ON COLUMN world_scene_definition.boundary_min_y_milli IS '人物中心可到达矩形的最小 Y，单位为千分之一场景格，包含端点';
COMMENT ON COLUMN world_scene_definition.boundary_max_x_milli IS '人物中心可到达矩形的最大 X，单位为千分之一场景格，包含端点';
COMMENT ON COLUMN world_scene_definition.boundary_max_y_milli IS '人物中心可到达矩形的最大 Y，单位为千分之一场景格，包含端点';
COMMENT ON COLUMN world_scene_definition.boundary_last_update_reason IS '管理员最近一次调整场景矩形边界时填写的操作原因';
COMMENT ON COLUMN world_scene_definition.boundary_updated_by_admin_user_id IS '最近一次修改场景矩形边界的后台管理员 ID';

COMMIT;
