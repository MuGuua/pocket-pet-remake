-- 083_encounter_battle_rewards.sql
-- 地图暗雷遭遇战支持独立奖励；怪物自身奖励继续保留，最终结算为遭遇战奖励 + 本次编队中所有怪物奖励。

ALTER TABLE scene_wild_encounter
  ADD COLUMN IF NOT EXISTS rewards JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN scene_wild_encounter.rewards IS
  '该暗雷遭遇战胜利后的固定奖励 JSON，结构与怪物战斗奖励保存输入一致。';

ALTER TABLE monster_battle_reward
  ADD COLUMN IF NOT EXISTS attr_key VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_type;

-- 历史版本里 reward_type=gold 实际表示“铜币总数”，先迁移为 copper，之后 gold 表示真正的金币数量。
UPDATE monster_battle_reward
SET reward_type = 'copper'
WHERE reward_type = 'gold';

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_type
  CHECK (reward_type IN ('exp', 'item', 'gold', 'silver', 'copper', 'attr'));

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_attr_key;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_attr_key
  CHECK (
    attr_key IN ('', 'free_attr_points', 'strength', 'vitality', 'agility', 'mind', 'hp_max', 'atk', 'def', 'spd', 'mana')
  );

COMMENT ON COLUMN monster_battle_reward.attr_key IS
  'reward_type=attr 时写入的玩家属性字段；其他奖励类型保持为空。';
