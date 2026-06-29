-- 070_equipment_enhance_success_required_level_band.sql
-- 强化成功率按「穿戴等级段（每10级）+ 目标强化等级」二维配置。

ALTER TABLE equipment_enhance_success_config
  ADD COLUMN IF NOT EXISTS required_level_min INTEGER NOT NULL DEFAULT 1;

UPDATE equipment_enhance_success_config
SET required_level_min = 1
WHERE required_level_min IS NULL OR required_level_min <= 0;

ALTER TABLE equipment_enhance_success_config
  DROP CONSTRAINT IF EXISTS equipment_enhance_success_config_pkey;

ALTER TABLE equipment_enhance_success_config
  ADD PRIMARY KEY (target_level, required_level_min);

ALTER TABLE equipment_enhance_success_config
  DROP CONSTRAINT IF EXISTS chk_enhance_required_level_band;

ALTER TABLE equipment_enhance_success_config
  ADD CONSTRAINT chk_enhance_required_level_band
  CHECK (required_level_min >= 1 AND (required_level_min - 1) % 10 = 0);

COMMENT ON COLUMN equipment_enhance_success_config.required_level_min IS
  '穿戴等级段起点：1=1~10级，11=11~20级，21=21~30级…';

-- 为 11~91 级穿戴段生成默认种子：在 +1~+10 段基础上，每升一段穿戴等级成功率 -5%（下限 1%）。
INSERT INTO equipment_enhance_success_config (
  target_level, required_level_min, success_rate_pct, description, status
)
SELECT
  base.target_level,
  band.required_level_min,
  GREATEST(1, base.success_rate_pct - band.band_penalty_pct),
  '穿戴 ' || band.required_level_min || '~' || (band.required_level_min + 9) || ' 级 · 强化至 +' || base.target_level,
  1
FROM equipment_enhance_success_config base
CROSS JOIN (
  VALUES
    (11, 5),
    (21, 10),
    (31, 15),
    (41, 20),
    (51, 25),
    (61, 30),
    (71, 35),
    (81, 40),
    (91, 45)
) AS band(required_level_min, band_penalty_pct)
WHERE base.required_level_min = 1
ON CONFLICT (target_level, required_level_min) DO UPDATE SET
  success_rate_pct = EXCLUDED.success_rate_pct,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
