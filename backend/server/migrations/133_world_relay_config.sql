BEGIN;
-- 默认关闭；只启用世界实体跨实例展示，不代表战斗/PVP/会话已经全面分布式。
ALTER TABLE world_aoi_config ADD COLUMN IF NOT EXISTS relay_enabled boolean NOT NULL DEFAULT false;
COMMENT ON COLUMN world_aoi_config.relay_enabled IS '世界实体 Redis Stream 广播开关，保存后所有参与实例重启生效';
COMMIT;
-- 回滚先关闭并重启旧版本服务，再由维护者移除 relay_enabled 列。
