-- Add the mana attribute required by the richer turn-battle damage formula.
-- The current battle MVP still exposes only HP/ATK/DEF/SPD to the client, so
-- mana is introduced server-side first and used by formula calculation only.

ALTER TABLE player_pet
ADD COLUMN IF NOT EXISTS mana INTEGER NOT NULL DEFAULT 1;

-- Backfill the demo pets so local PostgreSQL battles keep matching the
-- in-memory test stub after this migration is applied to an existing database.
UPDATE player_pet
SET mana = CASE id
  WHEN 20001 THEN 16
  WHEN 20002 THEN 20
  WHEN 20003 THEN 12
  ELSE GREATEST(1, level * 3)
END
WHERE mana <= 1;
