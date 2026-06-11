-- Add persistent player combat stats so the upcoming solo-character PVE flow
-- can load authoritative character attributes from storage instead of using
-- only in-memory battle templates.

ALTER TABLE player
ADD COLUMN IF NOT EXISTS energy INTEGER NOT NULL DEFAULT 100,
ADD COLUMN IF NOT EXISTS energy_max INTEGER NOT NULL DEFAULT 100,
ADD COLUMN IF NOT EXISTS atk INTEGER NOT NULL DEFAULT 24,
ADD COLUMN IF NOT EXISTS def INTEGER NOT NULL DEFAULT 12,
ADD COLUMN IF NOT EXISTS spd INTEGER NOT NULL DEFAULT 18,
ADD COLUMN IF NOT EXISTS mana INTEGER NOT NULL DEFAULT 20,
ADD COLUMN IF NOT EXISTS hit_pct INTEGER NOT NULL DEFAULT 10,
ADD COLUMN IF NOT EXISTS dodge_pct INTEGER NOT NULL DEFAULT 6,
ADD COLUMN IF NOT EXISTS crit_rate_pct INTEGER NOT NULL DEFAULT 10,
ADD COLUMN IF NOT EXISTS crit_dmg_pct INTEGER NOT NULL DEFAULT 155,
ADD COLUMN IF NOT EXISTS physical_resist_pct INTEGER NOT NULL DEFAULT 6,
ADD COLUMN IF NOT EXISTS skill_resist_pct INTEGER NOT NULL DEFAULT 4,
ADD COLUMN IF NOT EXISTS confusion_resist_pct INTEGER NOT NULL DEFAULT 8,
ADD COLUMN IF NOT EXISTS sleep_resist_pct INTEGER NOT NULL DEFAULT 8,
ADD COLUMN IF NOT EXISTS paralysis_resist_pct INTEGER NOT NULL DEFAULT 6,
ADD COLUMN IF NOT EXISTS seal_resist_pct INTEGER NOT NULL DEFAULT 6,
ADD COLUMN IF NOT EXISTS curse_resist_pct INTEGER NOT NULL DEFAULT 5,
ADD COLUMN IF NOT EXISTS crit_resist_pct INTEGER NOT NULL DEFAULT 4,
ADD COLUMN IF NOT EXISTS crit_dmg_resist_pct INTEGER NOT NULL DEFAULT 10,
ADD COLUMN IF NOT EXISTS character_resist_pct INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS pet_resist_pct INTEGER NOT NULL DEFAULT 4,
ADD COLUMN IF NOT EXISTS mercenary_resist_pct INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS generic_shield_pct INTEGER NOT NULL DEFAULT 2;

-- Keep the seeded demo accounts aligned with the same starter values used by
-- the in-memory repository so storage modes behave consistently.
UPDATE player
SET energy = 100,
    energy_max = 100,
    atk = CASE id
      WHEN 10001 THEN 24
      WHEN 10002 THEN 23
      ELSE atk
    END,
    def = CASE id
      WHEN 10001 THEN 12
      WHEN 10002 THEN 11
      ELSE def
    END,
    spd = CASE id
      WHEN 10001 THEN 18
      WHEN 10002 THEN 17
      ELSE spd
    END,
    mana = CASE id
      WHEN 10001 THEN 20
      WHEN 10002 THEN 18
      ELSE mana
    END,
    hit_pct = 10,
    dodge_pct = 6,
    crit_rate_pct = 10,
    crit_dmg_pct = 155,
    physical_resist_pct = 6,
    skill_resist_pct = 4,
    confusion_resist_pct = 8,
    sleep_resist_pct = 8,
    paralysis_resist_pct = 6,
    seal_resist_pct = 6,
    curse_resist_pct = 5,
    crit_resist_pct = 4,
    crit_dmg_resist_pct = 10,
    character_resist_pct = 0,
    pet_resist_pct = 4,
    mercenary_resist_pct = 0,
    generic_shield_pct = 2
WHERE id IN (10001, 10002);
