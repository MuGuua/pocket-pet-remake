-- 038_monster_battle_reward.sql
-- 怪物战斗奖励配置：仅支持经验与物品，供 PVE 结算与后台配置消费。

CREATE TABLE IF NOT EXISTS monster_battle_reward (
  id BIGSERIAL PRIMARY KEY,
  monster_id INTEGER NOT NULL,
  reward_type VARCHAR(16) NOT NULL,
  exp_target VARCHAR(16) NOT NULL DEFAULT 'player',
  item_id BIGINT NOT NULL DEFAULT 0,
  quantity BIGINT NOT NULL DEFAULT 0,
  exp_value BIGINT NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_monster_battle_reward_type CHECK (reward_type IN ('exp', 'item')),
  CONSTRAINT ck_monster_battle_reward_exp_target CHECK (exp_target IN ('player', 'pet')),
  CONSTRAINT ck_monster_battle_reward_quantity CHECK (quantity >= 0),
  CONSTRAINT ck_monster_battle_reward_exp_value CHECK (exp_value >= 0)
);

CREATE INDEX IF NOT EXISTS idx_monster_battle_reward_monster_id
  ON monster_battle_reward (monster_id, sort_order ASC, id ASC);

CREATE TRIGGER trg_monster_battle_reward_updated_at
BEFORE UPDATE ON monster_battle_reward
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 种子：沿用旧硬编码掉落与经验口径（9001/9002）。
INSERT INTO monster_battle_reward (monster_id, reward_type, exp_target, exp_value, item_id, quantity, sort_order, status)
VALUES
  (9001, 'exp', 'player', 28, 0, 0, 1, 1),
  (9001, 'exp', 'pet', 28, 0, 0, 2, 1),
  (9001, 'item', 'player', 0, 3101, 1, 3, 1),
  (9002, 'exp', 'player', 36, 0, 0, 1, 1),
  (9002, 'exp', 'pet', 36, 0, 0, 2, 1),
  (9002, 'item', 'player', 0, 3102, 1, 3, 1);
