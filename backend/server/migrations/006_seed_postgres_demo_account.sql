-- Seed the PostgreSQL demo account so `postgres_redis` mode can log in with
-- the same credentials and starter data as the in-memory development mode.
-- This keeps local testing consistent after switching the repository mode.

INSERT INTO account (
  id,
  account_name,
  password_hash,
  status
) VALUES (
  1,
  'demo',
  'd3ad9315b7be5dd53b31a273b3b3aba5defe700808305aa16a3062b76658a791',
  1
)
ON CONFLICT (id) DO UPDATE SET
  account_name = EXCLUDED.account_name,
  password_hash = EXCLUDED.password_hash,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- The player profile mirrors the memory repository defaults so world login,
-- position sync, and starter resources stay aligned across storage backends.
INSERT INTO player (
  id,
  account_id,
  name,
  level,
  exp,
  gold,
  scene_id,
  pos_x,
  pos_y,
  hp,
  hp_max,
  status
) VALUES (
  10001,
  1,
  'DemoTrainer',
  1,
  0,
  100,
  1,
  8,
  6,
  100,
  100,
  1
)
ON CONFLICT (id) DO UPDATE SET
  account_id = EXCLUDED.account_id,
  name = EXCLUDED.name,
  level = EXCLUDED.level,
  exp = EXCLUDED.exp,
  gold = EXCLUDED.gold,
  scene_id = EXCLUDED.scene_id,
  pos_x = EXCLUDED.pos_x,
  pos_y = EXCLUDED.pos_y,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- Starter pets are copied from the memory pet repository so battle, lineup,
-- and pet list flows behave the same after moving to PostgreSQL mode.
INSERT INTO player_pet (
  id,
  player_id,
  pet_id,
  level,
  exp,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  skill_ids
) VALUES
  (
    20001,
    10001,
    101,
    5,
    120,
    1,
    32,
    32,
    14,
    10,
    12,
    '[1001, 1002]'::jsonb
  ),
  (
    20002,
    10001,
    102,
    4,
    80,
    1,
    28,
    30,
    12,
    11,
    9,
    '[1001, 1003]'::jsonb
  ),
  (
    20003,
    10001,
    101,
    3,
    40,
    1,
    24,
    24,
    10,
    8,
    11,
    '[1001]'::jsonb
  )
ON CONFLICT (id) DO UPDATE SET
  player_id = EXCLUDED.player_id,
  pet_id = EXCLUDED.pet_id,
  level = EXCLUDED.level,
  exp = EXCLUDED.exp,
  quality = EXCLUDED.quality,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  skill_ids = EXCLUDED.skill_ids,
  updated_at = CURRENT_TIMESTAMP;

-- The first two pets are placed into the active lineup to match the
-- development-mode battle defaults and avoid an empty lineup after login.
INSERT INTO player_lineup (
  player_id,
  slot_index,
  pet_uid
) VALUES
  (10001, 0, 20001),
  (10001, 1, 20002)
ON CONFLICT (player_id, slot_index) DO UPDATE SET
  pet_uid = EXCLUDED.pet_uid,
  updated_at = CURRENT_TIMESTAMP;
