BEGIN;

-- P0-06 静态通行位图启用后，快速传送中心必须落在对应场景的已发布可通行格。
-- 这里只修复数据库权威的快速传送点；普通门和默认出生点由服务端世界拓扑代码同步维护。
UPDATE world_map_teleport_node
SET center_x = CASE scene_id
        WHEN 13 THEN 6
        WHEN 26 THEN 6
    END,
    center_y = CASE scene_id
        WHEN 13 THEN 8
        WHEN 26 THEN 6
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE scene_id IN (13, 26)
  AND (
      (scene_id = 13 AND (center_x IS DISTINCT FROM 6 OR center_y IS DISTINCT FROM 8))
      OR (scene_id = 26 AND (center_x IS DISTINCT FROM 6 OR center_y IS DISTINCT FROM 6))
  );

COMMIT;

-- 回滚说明：如需恢复旧值，请在确认旧坐标仍可通行后手动执行：
-- UPDATE world_map_teleport_node SET center_x = 6, center_y = 7 WHERE scene_id IN (13, 26);
