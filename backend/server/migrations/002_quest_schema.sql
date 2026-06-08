CREATE TABLE quest_template (
  id BIGSERIAL PRIMARY KEY,
  quest_id BIGINT NOT NULL,
  quest_type VARCHAR(16) NOT NULL,
  name VARCHAR(64) NOT NULL,
  title VARCHAR(128) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  chapter INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  accept_mode VARCHAR(16) NOT NULL DEFAULT 'AUTO',
  submit_mode VARCHAR(16) NOT NULL DEFAULT 'AUTO',
  can_abandon BOOLEAN NOT NULL DEFAULT FALSE,
  is_repeatable BOOLEAN NOT NULL DEFAULT FALSE,
  auto_track BOOLEAN NOT NULL DEFAULT TRUE,
  start_npc_id BIGINT NOT NULL DEFAULT 0,
  submit_npc_id BIGINT NOT NULL DEFAULT 0,
  mutually_exclusive_group VARCHAR(64) NOT NULL DEFAULT '',
  min_player_level INTEGER NOT NULL DEFAULT 1,
  visible_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
  unlock_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
  accept_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
  pre_quest_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  objectives_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  rewards_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  time_window_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  version INTEGER NOT NULL DEFAULT 1,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_quest_template_quest_id UNIQUE (quest_id)
);

CREATE INDEX idx_quest_template_type_status ON quest_template (quest_type, status);
CREATE INDEX idx_quest_template_sort_order ON quest_template (chapter, sort_order, quest_id);

CREATE TABLE player_quest (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  quest_id BIGINT NOT NULL,
  quest_type VARCHAR(16) NOT NULL,
  state VARCHAR(32) NOT NULL DEFAULT 'LOCKED',
  tracked BOOLEAN NOT NULL DEFAULT FALSE,
  progress_version BIGINT NOT NULL DEFAULT 0,
  accept_count INTEGER NOT NULL DEFAULT 0,
  accepted_at TIMESTAMPTZ NULL,
  completed_at TIMESTAMPTZ NULL,
  submitted_at TIMESTAMPTZ NULL,
  expire_at TIMESTAMPTZ NULL,
  next_refresh_at TIMESTAMPTZ NULL,
  last_event_at TIMESTAMPTZ NULL,
  reward_claimed BOOLEAN NOT NULL DEFAULT FALSE,
  reward_version INTEGER NOT NULL DEFAULT 0,
  ext_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_quest UNIQUE (player_id, quest_id)
);

CREATE INDEX idx_player_quest_player_state ON player_quest (player_id, state);
CREATE INDEX idx_player_quest_player_type ON player_quest (player_id, quest_type);
CREATE INDEX idx_player_quest_player_tracked ON player_quest (player_id, tracked);
CREATE INDEX idx_player_quest_expire_at ON player_quest (expire_at);
CREATE INDEX idx_player_quest_next_refresh_at ON player_quest (next_refresh_at);

CREATE TABLE player_quest_objective (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  quest_id BIGINT NOT NULL,
  objective_id BIGINT NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  current_value BIGINT NOT NULL DEFAULT 0,
  target_value BIGINT NOT NULL DEFAULT 0,
  completed BOOLEAN NOT NULL DEFAULT FALSE,
  target_selector_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  guide_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_quest_objective UNIQUE (player_id, quest_id, objective_id)
);

CREATE INDEX idx_player_quest_objective_player_quest ON player_quest_objective (player_id, quest_id);
CREATE INDEX idx_player_quest_objective_player_completed ON player_quest_objective (player_id, completed);

CREATE TABLE player_quest_event_log (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL,
  quest_id BIGINT NOT NULL DEFAULT 0,
  objective_id BIGINT NOT NULL DEFAULT 0,
  event_type VARCHAR(32) NOT NULL,
  event_key VARCHAR(128) NOT NULL DEFAULT '',
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_player_quest_event_log_player_time ON player_quest_event_log (player_id, created_at DESC);
CREATE INDEX idx_player_quest_event_log_player_type ON player_quest_event_log (player_id, event_type);

CREATE TRIGGER set_quest_template_updated_at
BEFORE UPDATE ON quest_template
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_quest_updated_at
BEFORE UPDATE ON player_quest
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_player_quest_objective_updated_at
BEFORE UPDATE ON player_quest_objective
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
