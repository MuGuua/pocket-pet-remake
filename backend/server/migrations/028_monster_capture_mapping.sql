-- 028_monster_capture_mapping.sql
-- 怪物模板增加捕捉配置：捕捉成功后发放关联系统宠物，不继承战斗数值。

ALTER TABLE monster_definition
  ADD COLUMN IF NOT EXISTS is_capturable INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS capture_pet_id INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS capture_rate_base INTEGER NOT NULL DEFAULT 5000,
  ADD COLUMN IF NOT EXISTS capture_min_hp_pct INTEGER NOT NULL DEFAULT 30,
  ADD COLUMN IF NOT EXISTS capture_item_ids JSONB NOT NULL DEFAULT '[2001]'::jsonb;

COMMENT ON COLUMN monster_definition.is_capturable IS '是否可被捕捉：1=可捕捉，0=不可捕捉';
COMMENT ON COLUMN monster_definition.capture_pet_id IS '捕捉成功后发放的系统宠物模板 pet_id';
COMMENT ON COLUMN monster_definition.capture_rate_base IS '基础捕捉成功率（万分比，后续战斗捕捉公式使用）';
COMMENT ON COLUMN monster_definition.capture_min_hp_pct IS '允许尝试捕捉的敌方最低生命百分比';
COMMENT ON COLUMN monster_definition.capture_item_ids IS '允许用于捕捉的道具 item_id 列表';

-- 9001 野生怪物可捕捉，成功后发放 pet_id=103 的野外捕捉宠物模板
UPDATE monster_definition
SET
  is_capturable = 1,
  capture_pet_id = 103,
  capture_rate_base = 5000,
  capture_min_hp_pct = 30,
  capture_item_ids = '[2001]'::jsonb,
  updated_at = CURRENT_TIMESTAMP
WHERE monster_id = 9001;

UPDATE monster_definition
SET
  is_capturable = 0,
  capture_pet_id = 0,
  updated_at = CURRENT_TIMESTAMP
WHERE monster_id = 9002;
