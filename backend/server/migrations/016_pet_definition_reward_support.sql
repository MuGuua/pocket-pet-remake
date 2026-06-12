CREATE TABLE IF NOT EXISTS pet_definition (
  pet_id INTEGER PRIMARY KEY,
  pet_name VARCHAR(64) NOT NULL DEFAULT '',
  level INTEGER NOT NULL DEFAULT 1,
  quality INTEGER NOT NULL DEFAULT 1,
  hp INTEGER NOT NULL DEFAULT 1,
  hp_max INTEGER NOT NULL DEFAULT 1,
  atk INTEGER NOT NULL DEFAULT 1,
  def INTEGER NOT NULL DEFAULT 1,
  spd INTEGER NOT NULL DEFAULT 1,
  mana INTEGER NOT NULL DEFAULT 0,
  skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  status INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_pet_definition_updated_at
BEFORE UPDATE ON pet_definition
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO pet_definition (
  pet_id,
  pet_name,
  level,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  skill_ids,
  status
) VALUES
  (
    101,
    '嫩叶犬',
    5,
    1,
    32,
    32,
    14,
    10,
    12,
    16,
    '[1001, 1002]'::jsonb,
    1
  ),
  (
    102,
    '潮汐狐',
    4,
    1,
    28,
    30,
    12,
    11,
    9,
    20,
    '[1001, 1003]'::jsonb,
    1
  )
ON CONFLICT (pet_id) DO UPDATE SET
  pet_name = EXCLUDED.pet_name,
  level = EXCLUDED.level,
  quality = EXCLUDED.quality,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  mana = EXCLUDED.mana,
  skill_ids = EXCLUDED.skill_ids,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

UPDATE quest_template
SET rewards_json = '[
  { "type": "gold", "value": 150 },
  { "type": "item", "item_id": 2001, "count": 2, "value": 0 },
  { "type": "pet", "pet_id": 102, "value": 0 }
]'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE quest_id = 1002;
