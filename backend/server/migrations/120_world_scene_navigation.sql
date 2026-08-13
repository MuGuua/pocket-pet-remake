BEGIN;

-- 场景静态通行位图采用版本化存储。数据库中的已发布版本是服务端移动判定的唯一事实来源，
-- Godot 导出工具只生成草稿数据或待执行 SQL，不直接修改正式数据库。
CREATE TABLE IF NOT EXISTS world_scene_navigation (
    navigation_id BIGSERIAL PRIMARY KEY,
    scene_id INTEGER NOT NULL REFERENCES world_scene_definition(scene_id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    origin_x_milli INTEGER NOT NULL,
    origin_y_milli INTEGER NOT NULL,
    grid_width INTEGER NOT NULL,
    grid_height INTEGER NOT NULL,
    cell_size_milli INTEGER NOT NULL,
    navigation_data BYTEA NOT NULL,
    data_hash CHAR(64) NOT NULL,
    walkable_cell_count INTEGER NOT NULL,
    source_scene_path VARCHAR(512) NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 2,
    change_reason VARCHAR(500) NOT NULL,
    publish_reason VARCHAR(500) NOT NULL DEFAULT '',
    created_by_admin_user_id BIGINT REFERENCES admin_user(id),
    published_by_admin_user_id BIGINT REFERENCES admin_user(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_world_scene_navigation_scene_version UNIQUE (scene_id, version),
    CONSTRAINT ck_world_scene_navigation_version CHECK (version > 0),
    CONSTRAINT ck_world_scene_navigation_origin CHECK (
        origin_x_milli BETWEEN -10000000 AND 10000000
        AND origin_y_milli BETWEEN -10000000 AND 10000000
    ),
    CONSTRAINT ck_world_scene_navigation_grid_size CHECK (
        grid_width BETWEEN 1 AND 4096
        AND grid_height BETWEEN 1 AND 4096
        AND (grid_width::BIGINT * grid_height::BIGINT) <= 4194304
    ),
    CONSTRAINT ck_world_scene_navigation_cell_size CHECK (cell_size_milli BETWEEN 1 AND 100000),
    CONSTRAINT ck_world_scene_navigation_data_length CHECK (
        OCTET_LENGTH(navigation_data) = ((grid_width::BIGINT * grid_height::BIGINT + 7) / 8)
    ),
    CONSTRAINT ck_world_scene_navigation_hash CHECK (data_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT ck_world_scene_navigation_walkable_count CHECK (
        walkable_cell_count BETWEEN 0 AND (grid_width::BIGINT * grid_height::BIGINT)
    ),
    CONSTRAINT ck_world_scene_navigation_status CHECK (status IN (0, 1, 2)),
    CONSTRAINT ck_world_scene_navigation_reason CHECK (
        CHAR_LENGTH(BTRIM(change_reason)) BETWEEN 1 AND 500
        AND CHAR_LENGTH(BTRIM(publish_reason)) <= 500
        AND (status <> 1 OR CHAR_LENGTH(BTRIM(publish_reason)) BETWEEN 1 AND 500)
    )
);

-- PostgreSQL 部分唯一索引确保任意时刻每张场景最多只有一个已发布版本。
CREATE UNIQUE INDEX IF NOT EXISTS uk_world_scene_navigation_published_scene
    ON world_scene_navigation(scene_id)
    WHERE status = 1;

CREATE INDEX IF NOT EXISTS idx_world_scene_navigation_scene_versions
    ON world_scene_navigation(scene_id, version DESC);

CREATE INDEX IF NOT EXISTS idx_world_scene_navigation_status
    ON world_scene_navigation(status, scene_id);

COMMENT ON TABLE world_scene_navigation IS '场景静态通行位图版本；0=历史，1=已发布，2=草稿';
COMMENT ON COLUMN world_scene_navigation.origin_x_milli IS '位图左上角第一个单元格的 X 原点，单位为千分之一场景格';
COMMENT ON COLUMN world_scene_navigation.origin_y_milli IS '位图左上角第一个单元格的 Y 原点，单位为千分之一场景格';
COMMENT ON COLUMN world_scene_navigation.cell_size_milli IS '每个位图单元格边长，单位为千分之一场景格';
COMMENT ON COLUMN world_scene_navigation.navigation_data IS '按行优先、高位优先编码的位图；1 表示人物碰撞体可站立，0 表示阻挡';
COMMENT ON COLUMN world_scene_navigation.data_hash IS '服务端根据原始位图字节计算的 SHA-256 十六进制摘要';
COMMENT ON COLUMN world_scene_navigation.source_scene_path IS '生成位图的 Godot 场景资源路径，仅用于审计和问题定位';

COMMIT;

-- 回滚说明：确认没有服务端进程依赖该表后执行：
-- DROP TABLE IF EXISTS world_scene_navigation;
