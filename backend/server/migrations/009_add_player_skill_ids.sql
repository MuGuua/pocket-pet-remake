-- Persist player-side battle skills so solo-character PVE can load the same
-- authoritative skill list across reconnect, world snapshot, and battle start.

ALTER TABLE player
ADD COLUMN IF NOT EXISTS skill_ids JSONB NOT NULL DEFAULT '[1101, 1001]'::jsonb;

-- Seed the demo accounts with one人物主动技能 + 普攻，保持测试仓储与 PostgreSQL
-- 仓储在技能顺序上的一致性，让移动端进入战斗后优先看到人物专属技能。
UPDATE player
SET skill_ids = '[1101, 1001]'::jsonb
WHERE id IN (10001, 10002);
