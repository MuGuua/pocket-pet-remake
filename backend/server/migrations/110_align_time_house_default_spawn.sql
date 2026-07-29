-- 110_align_time_house_default_spawn.sql
-- 多人同屏出生坐标统一改由服务端权威下发后，注册默认出生格从时光小屋 (4,4) 对齐为
-- 客户端地图调好的 (6,6)。此处把仍停留在旧默认格且从未移动过的存量角色一并对齐，
-- 避免这些角色在其他玩家视角出现在未经调校的旧坐标上。

UPDATE player
SET pos_x = 6,
    pos_y = 6
WHERE scene_id = 7
  AND pos_x = 4
  AND pos_y = 4;
