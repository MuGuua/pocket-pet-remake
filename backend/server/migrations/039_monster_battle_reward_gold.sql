-- 039_monster_battle_reward_gold.sql
-- 怪物战斗奖励新增铜币类型；铜币数量写入 exp_value 字段。

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_type;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_type
  CHECK (reward_type IN ('exp', 'item', 'gold'));
