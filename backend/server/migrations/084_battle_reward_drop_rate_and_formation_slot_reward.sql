-- 084_battle_reward_drop_rate_and_formation_slot_reward.sql
-- 战斗奖励增加逐项掉落率；暗雷编队槽位支持单独关闭怪物自身奖励。

ALTER TABLE monster_battle_reward
  ADD COLUMN IF NOT EXISTS drop_rate INTEGER NOT NULL DEFAULT 10000;

ALTER TABLE monster_battle_reward
  DROP CONSTRAINT IF EXISTS ck_monster_battle_reward_drop_rate;

ALTER TABLE monster_battle_reward
  ADD CONSTRAINT ck_monster_battle_reward_drop_rate
  CHECK (drop_rate BETWEEN 1 AND 10000);

COMMENT ON COLUMN monster_battle_reward.drop_rate IS
  '奖励掉落率，万分比。10000=100%，1=0.01%。';

UPDATE scene_wild_encounter
SET formations = normalized.formations
FROM (
  SELECT
    scene_id,
    jsonb_agg(
      jsonb_set(
        formation,
        '{monster_slots}',
        (
          SELECT jsonb_agg(jsonb_build_object('monster_id', monster_id::integer, 'reward_enabled', true))
          FROM jsonb_array_elements_text(COALESCE(formation->'spawn_monster_ids', '[]'::jsonb)) AS monster_id
        ),
        true
      )
    ) AS formations
  FROM scene_wild_encounter swe
  CROSS JOIN LATERAL jsonb_array_elements(COALESCE(swe.formations, '[]'::jsonb)) AS formation
  WHERE jsonb_typeof(COALESCE(swe.formations, '[]'::jsonb)) = 'array'
    AND NOT (formation ? 'monster_slots')
  GROUP BY scene_id
) AS normalized
WHERE scene_wild_encounter.scene_id = normalized.scene_id;
