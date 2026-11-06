BEGIN;

-- 仅由维护者手动执行。默认关闭，先保持现有小地图全场景可见；开启前核对实际镜头和预加载余量。
CREATE TABLE IF NOT EXISTS world_aoi_config (
    config_id smallint PRIMARY KEY CHECK (config_id = 1),
    enabled boolean NOT NULL DEFAULT false,
    cell_size_milli integer NOT NULL CHECK (cell_size_milli BETWEEN 1000 AND 1000000),
    neighbor_radius integer NOT NULL CHECK (neighbor_radius BETWEEN 1 AND 4),
    change_reason varchar(500) NOT NULL CHECK (length(btrim(change_reason)) > 0),
    updated_by_admin_user_id bigint REFERENCES admin_user(id),
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO world_aoi_config(config_id,enabled,cell_size_milli,neighbor_radius,change_reason)
VALUES(1,false,24000,1,'初始化 AOI 配置，保持关闭；启用前核对镜头覆盖并重启服务')
ON CONFLICT (config_id) DO NOTHING;
COMMENT ON TABLE world_aoi_config IS '进程启动时加载的玩家网格可见范围配置；修改后重启生效';
COMMENT ON COLUMN world_aoi_config.cell_size_milli IS '网格边长，千分之一场景格；邻域必须覆盖镜头及预加载余量';
COMMENT ON COLUMN world_aoi_config.neighbor_radius IS '以当前格为中心的切比雪夫邻域半径';

COMMIT;

-- 回滚：先在后台关闭 AOI 并重启旧版本兼容服务；确认没有新代码读取后，由维护者手动 DROP TABLE world_aoi_config。
