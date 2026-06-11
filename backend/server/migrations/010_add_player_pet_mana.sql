-- Persist pet mana so lineup loading, battle actor assembly, and reconnect
-- snapshots all read one authoritative resource field from PostgreSQL.

ALTER TABLE player_pet
ADD COLUMN IF NOT EXISTS mana INTEGER NOT NULL DEFAULT 0;

-- Keep the seeded demo pets aligned with the in-memory repository so local
-- PostgreSQL mode and test fixtures expose the same starter mana values.
UPDATE player_pet
SET mana = CASE id
  WHEN 20001 THEN 12
  WHEN 20002 THEN 18
  WHEN 20003 THEN 10
  ELSE mana
END
WHERE id IN (20001, 20002, 20003);
