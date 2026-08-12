-- 配置服务端权威世界移动速度和弱网容差；迁移只创建并写入初始正式配置，由用户手动执行。
CREATE TABLE IF NOT EXISTS world_movement_config (
    config_id SMALLINT PRIMARY KEY,
    speed_milli_cells_per_second INTEGER NOT NULL,
    max_elapsed_ms INTEGER NOT NULL,
    axis_tolerance_milli INTEGER NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    last_update_reason VARCHAR(500) NOT NULL DEFAULT '初始化权威移动配置',
    updated_by_admin_user_id BIGINT NULL REFERENCES admin_user(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_world_movement_config_singleton CHECK (config_id = 1),
    CONSTRAINT ck_world_movement_config_speed CHECK (speed_milli_cells_per_second > 0),
    CONSTRAINT ck_world_movement_config_elapsed CHECK (max_elapsed_ms BETWEEN 50 AND 2000),
    CONSTRAINT ck_world_movement_config_axis_tolerance CHECK (axis_tolerance_milli BETWEEN 0 AND 1000),
    CONSTRAINT ck_world_movement_config_status CHECK (status IN (0, 1))
);

COMMENT ON TABLE world_movement_config IS '服务端权威世界移动配置，坐标和速度使用千分之一场景格定点整数';
COMMENT ON COLUMN world_movement_config.speed_milli_cells_per_second IS '每秒允许移动的千分之一场景格，3750 对应每秒 3.75 格';
COMMENT ON COLUMN world_movement_config.max_elapsed_ms IS '单次移动计算允许采用的最大服务端时间跨度，防止断流后一次性跳跃';
COMMENT ON COLUMN world_movement_config.axis_tolerance_milli IS '四方向移动时非主轴候选坐标允许的最大千分之一格误差';
COMMENT ON COLUMN world_movement_config.last_update_reason IS '管理员最近一次调整配置时填写的操作原因';
COMMENT ON COLUMN world_movement_config.updated_by_admin_user_id IS '最近一次修改配置的后台管理员 ID';

INSERT INTO world_movement_config (
    config_id,
    speed_milli_cells_per_second,
    max_elapsed_ms,
    axis_tolerance_milli,
    status
) VALUES (1, 3750, 300, 125, 1)
ON CONFLICT (config_id) DO NOTHING;
