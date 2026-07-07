CREATE TABLE IF NOT EXISTS player_equipment_snapshot (
    player_id BIGINT NOT NULL REFERENCES player(id) ON DELETE CASCADE,
    equip_slot TEXT NOT NULL,
    item_uid TEXT NOT NULL DEFAULT '',
    item_id BIGINT NOT NULL DEFAULT 0,
    item_name TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    required_level INTEGER NOT NULL DEFAULT 0,
    enhance_level INTEGER NOT NULL DEFAULT 0,
    is_damaged BOOLEAN NOT NULL DEFAULT FALSE,
    appearance_skin_id TEXT NOT NULL DEFAULT '',
    appearance_only BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    bonus_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    weapon_skills_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    weapon_type TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, equip_slot)
);

CREATE INDEX IF NOT EXISTS idx_player_equipment_snapshot_player_id
    ON player_equipment_snapshot (player_id);

CREATE TABLE IF NOT EXISTS player_skill_progress_snapshot (
    player_id BIGINT NOT NULL REFERENCES player(id) ON DELETE CASCADE,
    skill_id INTEGER NOT NULL,
    skill_exp INTEGER NOT NULL DEFAULT 0,
    skill_level INTEGER NOT NULL DEFAULT 1,
    is_learned BOOLEAN NOT NULL DEFAULT FALSE,
    learned_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_player_skill_progress_snapshot_player_id
    ON player_skill_progress_snapshot (player_id);
