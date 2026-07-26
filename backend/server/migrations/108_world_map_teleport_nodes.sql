BEGIN;

-- 世界地图快速传送配置由服务端权威读取；客户端只提交目标 scene_id，不能指定出生坐标。
CREATE TABLE IF NOT EXISTS world_map_teleport_node (
    scene_id INTEGER PRIMARY KEY REFERENCES world_scene_definition(scene_id),
    center_x INTEGER NOT NULL,
    center_y INTEGER NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE world_map_teleport_node IS '世界地图快速传送节点及目标地图中心出生格配置';
COMMENT ON COLUMN world_map_teleport_node.scene_id IS '目标世界场景 ID，同时保证每张地图最多一个快速传送中心';
COMMENT ON COLUMN world_map_teleport_node.center_x IS '服务端场景坐标系中的中心出生格 X';
COMMENT ON COLUMN world_map_teleport_node.center_y IS '服务端场景坐标系中的中心出生格 Y';
COMMENT ON COLUMN world_map_teleport_node.status IS '0=停用，1=允许快速传送';

-- 中心格按 Godot 地图实际碰撞边界换算；时光小屋和未落地的闪光平原不开放快速传送。
INSERT INTO world_map_teleport_node (scene_id, center_x, center_y, status)
VALUES
    (1, 5, 7, 1),
    (2, 5, 5, 1),
    (3, 7, 7, 1),
    (4, 4, 5, 1),
    (5, 6, 6, 1),
    (6, 5, 5, 1)
ON CONFLICT (scene_id) DO UPDATE
SET center_x = EXCLUDED.center_x,
    center_y = EXCLUDED.center_y,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

COMMIT;
