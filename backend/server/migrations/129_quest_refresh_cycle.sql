-- 129_quest_refresh_cycle.sql
-- 为可重复任务增加刷新周期配置，并让 can_abandon / is_repeatable 真正生效。
--
-- 背景：quest_template 早有 can_abandon、is_repeatable 两列，但服务端从未读取，
-- 客户端「日常任务」页签的任务交付后永远停留在已完成，无法每日重做；玩家也无法放弃任务。
--
-- refresh_cycle 取值：
--   none   —— 不按时间刷新；is_repeatable = TRUE 时交付后立即可再次接取。
--   daily  —— 每天 refresh_hour 点刷新（服务器时区）。
--   weekly —— 每周一 refresh_hour 点刷新（服务器时区）。
-- 只有 is_repeatable = TRUE 的任务才会被刷新；refresh_cycle 对不可重复任务无效。
--
-- 刷新语义：到达 player_quest.next_refresh_at 后，已完成的任务重置为可接取状态，
-- 并清空目标进度；completed_at 保留，因此依赖它的后置任务不会被重新锁住。

ALTER TABLE quest_template
  ADD COLUMN IF NOT EXISTS refresh_cycle VARCHAR(16) NOT NULL DEFAULT 'none',
  ADD COLUMN IF NOT EXISTS refresh_hour INTEGER NOT NULL DEFAULT 5;

ALTER TABLE quest_template DROP CONSTRAINT IF EXISTS ck_quest_template_refresh_cycle;
ALTER TABLE quest_template
  ADD CONSTRAINT ck_quest_template_refresh_cycle CHECK (refresh_cycle IN ('none', 'daily', 'weekly'));

ALTER TABLE quest_template DROP CONSTRAINT IF EXISTS ck_quest_template_refresh_hour;
ALTER TABLE quest_template
  ADD CONSTRAINT ck_quest_template_refresh_hour CHECK (refresh_hour >= 0 AND refresh_hour <= 23);

COMMENT ON COLUMN quest_template.refresh_cycle IS '可重复任务刷新周期：none / daily / weekly';
COMMENT ON COLUMN quest_template.refresh_hour IS '刷新时刻（0~23 点，服务器时区）';

-- 已有日常任务按「每日 5 点刷新、可重复」初始化，避免上线后日常仍然一次性。
UPDATE quest_template
SET is_repeatable = TRUE,
    refresh_cycle = 'daily'
WHERE UPPER(quest_type) = 'DAILY' AND refresh_cycle = 'none';
