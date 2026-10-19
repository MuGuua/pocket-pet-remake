-- 126_battle_reward_draw_group.sql
-- 把「权重抽取」下沉到 monster_battle_reward：新增 draw_group 与 weight。
-- draw_group = 0（默认）保持原有语义：该条独立按 drop_rate 判定，可与其他条目同时掉落。
-- draw_group > 0：同一怪物、同一组内所有启用条目按 weight 加权抽 1 个；不同组互不影响。
-- 这样单只怪既能配固定掉落，也能配「一池候选中必出 1 个」，无需额外 Boss 概念。

ALTER TABLE monster_battle_reward
  ADD COLUMN IF NOT EXISTS draw_group INTEGER NOT NULL DEFAULT 0;

ALTER TABLE monster_battle_reward
  ADD COLUMN IF NOT EXISTS weight INTEGER NOT NULL DEFAULT 100;

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_draw_group;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_draw_group
  CHECK (draw_group >= 0);

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_weight;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_weight
  CHECK (weight > 0);

COMMENT ON COLUMN monster_battle_reward.draw_group IS
  '0 表示独立判定（按 drop_rate，可与其他条目同时掉落）；大于 0 表示权重抽取组，同组内按 weight 加权抽 1 个，不同组互不影响。';

COMMENT ON COLUMN monster_battle_reward.weight IS
  'draw_group > 0 时生效的抽取权重；同组内权重越大越容易命中。draw_group = 0 时忽略。';

-- 按怪物 + 抽取组建立索引，抽取时按怪物读取后分组。
CREATE INDEX IF NOT EXISTS idx_monster_battle_reward_draw
  ON monster_battle_reward (monster_id, draw_group ASC, sort_order ASC, id ASC)
  WHERE status = 1;

-- 125 建立的场景级掉落池已被本方案取代：能力下沉到每只怪自己的奖励配置。
-- 该表已于 2026-10-19 确认 0 行后删除；服务端代码不再引用。
