CREATE TABLE IF NOT EXISTS player_feature_unlock (
  player_id BIGINT NOT NULL,
  feature_id BIGINT NOT NULL,
  reason_type VARCHAR(64) NOT NULL DEFAULT '',
  reason_ref_id BIGINT NOT NULL DEFAULT 0,
  operator_type VARCHAR(32) NOT NULL DEFAULT '',
  operator_id BIGINT NOT NULL DEFAULT 0,
  unlocked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, feature_id)
);

CREATE INDEX IF NOT EXISTS idx_player_feature_unlock_player_id
ON player_feature_unlock (player_id);

UPDATE quest_template
SET rewards_json = '[
  { "type": "gold", "value": 150 },
  { "type": "item", "item_id": 2001, "count": 2, "value": 0 },
  { "type": "pet", "pet_id": 102, "value": 0 },
  { "type": "feature_unlock", "value": 1 }
]'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE quest_id = 1002;
