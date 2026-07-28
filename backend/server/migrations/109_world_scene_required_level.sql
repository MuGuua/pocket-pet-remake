-- 地图最低进入等级由数据库和运营后台维护；服务端在玩家切图前读取该配置并执行权威校验。
ALTER TABLE world_scene_definition
  ADD COLUMN IF NOT EXISTS required_level INTEGER NOT NULL DEFAULT 1;

-- 地图准入等级必须落在当前玩家等级体系支持的 1~100 级范围内。
ALTER TABLE world_scene_definition
  DROP CONSTRAINT IF EXISTS ck_world_scene_definition_required_level;

ALTER TABLE world_scene_definition
  ADD CONSTRAINT ck_world_scene_definition_required_level
  CHECK (required_level >= 1 AND required_level <= 100);

-- 本次上线不改变任何现有地图的进入条件，全部初始化为 1 级。
UPDATE world_scene_definition
SET required_level = 1,
    updated_at = CURRENT_TIMESTAMP
WHERE required_level IS DISTINCT FROM 1;

COMMENT ON COLUMN world_scene_definition.required_level IS
  '玩家进入该地图所需的最低角色等级；切图时由服务端使用玩家权威等级校验。';
