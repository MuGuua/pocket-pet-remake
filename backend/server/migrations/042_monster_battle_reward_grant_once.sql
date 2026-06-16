-- 042_monster_battle_reward_grant_once.sql
-- 怪物战斗奖励支持“唯一掉落”标记，并记录玩家已获得的唯一物品。

ALTER TABLE monster_battle_reward
  ADD COLUMN IF NOT EXISTS grant_once SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_grant_once;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_grant_once
  CHECK (grant_once IN (0, 1));

COMMENT ON COLUMN monster_battle_reward.grant_once IS
  '物品奖励是否唯一：1 表示玩家已获得后不再重复发放，0 表示可重复获得。';

CREATE TABLE IF NOT EXISTS player_unique_item_obtained (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  item_id BIGINT NOT NULL,
  first_reason_type VARCHAR(64) NOT NULL DEFAULT '',
  first_reason_ref_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_unique_item_obtained UNIQUE (player_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_player_unique_item_obtained_player_id
  ON player_unique_item_obtained (player_id);
