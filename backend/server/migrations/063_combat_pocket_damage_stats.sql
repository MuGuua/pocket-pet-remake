-- 063_combat_pocket_damage_stats.sql
-- 战斗单位口袋伤害扩展属性：守护、天赋增伤/减伤、元素克制/被克。

ALTER TABLE player
  ADD COLUMN IF NOT EXISTS guard INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_dmg_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_reduce_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_adv_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_penalty_pct INTEGER NOT NULL DEFAULT 0;

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS guard INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_dmg_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_reduce_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_adv_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_penalty_pct INTEGER NOT NULL DEFAULT 0;

ALTER TABLE monster_definition
  ADD COLUMN IF NOT EXISTS guard INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_dmg_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS talent_reduce_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_adv_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS element_penalty_pct INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN player.guard IS '守护；参与伤害分母，0 表示战斗时回退为有效防御面板';
COMMENT ON COLUMN player_pet.guard IS '守护；参与伤害分母，0 表示战斗时回退为有效防御面板';
COMMENT ON COLUMN monster_definition.guard IS '守护；参与伤害分母，0 表示战斗时回退为 def';

INSERT INTO pet_combat_stat_cap (stat_key, cap_value, description, status)
VALUES
  ('guard', 250000, '守护', 1),
  ('talent_dmg_pct', 200, '法宝天赋增伤（百分比）', 1),
  ('talent_reduce_pct', 100, '法宝天赋减伤（百分比）', 1),
  ('element_adv_pct', 100, '元素克制增伤（百分比）', 1),
  ('element_penalty_pct', 100, '元素被克减伤（百分比）', 1)
ON CONFLICT (stat_key) DO UPDATE SET
  cap_value = EXCLUDED.cap_value,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
