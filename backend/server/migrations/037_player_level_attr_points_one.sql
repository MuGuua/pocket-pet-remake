-- 037_player_level_attr_points_one.sql
-- 将 1~99 级升级奖励属性点统一调整为每级 1 点；100 级保持 0。

UPDATE player_level_config
SET attr_points = CASE
  WHEN level >= 100 THEN 0
  ELSE 1
END,
updated_at = CURRENT_TIMESTAMP
WHERE level BETWEEN 1 AND 100;
