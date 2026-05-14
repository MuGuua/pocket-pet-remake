CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE account (
  id BIGSERIAL PRIMARY KEY,
  account_name VARCHAR(64) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  last_login_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_account_name UNIQUE (account_name)
);

CREATE TABLE player (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL,
  name VARCHAR(64) NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  exp BIGINT NOT NULL DEFAULT 0,
  gold BIGINT NOT NULL DEFAULT 0,
  scene_id INTEGER NOT NULL DEFAULT 1,
  pos_x INTEGER NOT NULL DEFAULT 0,
  pos_y INTEGER NOT NULL DEFAULT 0,
  hp INTEGER NOT NULL DEFAULT 100,
  hp_max INTEGER NOT NULL DEFAULT 100,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_name UNIQUE (name)
);

CREATE INDEX idx_player_account_id ON player (account_id);

CREATE TABLE player_pet (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  pet_id INTEGER NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  exp BIGINT NOT NULL DEFAULT 0,
  quality INTEGER NOT NULL DEFAULT 1,
  hp INTEGER NOT NULL DEFAULT 1,
  hp_max INTEGER NOT NULL DEFAULT 1,
  atk INTEGER NOT NULL DEFAULT 1,
  def INTEGER NOT NULL DEFAULT 1,
  spd INTEGER NOT NULL DEFAULT 1,
  skill_ids JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_player_pet_player_id ON player_pet (player_id);

CREATE TABLE player_item (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  item_id INTEGER NOT NULL,
  count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_item UNIQUE (player_id, item_id)
);

CREATE INDEX idx_player_item_player_id ON player_item (player_id);

CREATE TABLE player_lineup (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  slot_index INTEGER NOT NULL,
  pet_uid BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_lineup_slot UNIQUE (player_id, slot_index),
  CONSTRAINT uk_player_lineup_pet UNIQUE (player_id, pet_uid)
);

CREATE TABLE battle_record (
  id BIGSERIAL PRIMARY KEY,
  battle_id BIGINT NOT NULL,
  player_id BIGINT NOT NULL,
  battle_type INTEGER NOT NULL,
  result SMALLINT NOT NULL,
  reward_gold BIGINT NOT NULL DEFAULT 0,
  reward_exp BIGINT NOT NULL DEFAULT 0,
  captured_pet_uid BIGINT NOT NULL DEFAULT 0,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_battle_record_battle_player UNIQUE (battle_id, player_id)
);

CREATE INDEX idx_battle_record_player_id ON battle_record (player_id);

CREATE TRIGGER set_account_updated_at
BEFORE UPDATE ON account
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_updated_at
BEFORE UPDATE ON player
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_pet_updated_at
BEFORE UPDATE ON player_pet
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_item_updated_at
BEFORE UPDATE ON player_item
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_lineup_updated_at
BEFORE UPDATE ON player_lineup
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
