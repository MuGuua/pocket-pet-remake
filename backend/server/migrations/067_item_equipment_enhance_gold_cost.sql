-- 067_item_equipment_enhance_gold_cost.sql
-- 每件装备模板独立配置强化铜币公式：基础值 + 每级固定递增或百分比复合递增。

ALTER TABLE item_equipment_extra
  ADD COLUMN IF NOT EXISTS enhance_gold_cost_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS enhance_gold_base_copper BIGINT NOT NULL DEFAULT 100,
  ADD COLUMN IF NOT EXISTS enhance_gold_increment_mode VARCHAR(16) NOT NULL DEFAULT 'fixed',
  ADD COLUMN IF NOT EXISTS enhance_gold_increment_fixed BIGINT NOT NULL DEFAULT 200,
  ADD COLUMN IF NOT EXISTS enhance_gold_increment_percent INTEGER NOT NULL DEFAULT 0;

ALTER TABLE item_equipment_extra
  DROP CONSTRAINT IF EXISTS chk_item_equipment_enhance_gold_base;
ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_enhance_gold_base CHECK (enhance_gold_base_copper >= 0);

ALTER TABLE item_equipment_extra
  DROP CONSTRAINT IF EXISTS chk_item_equipment_enhance_gold_mode;
ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_enhance_gold_mode CHECK (enhance_gold_increment_mode IN ('fixed', 'percent'));

ALTER TABLE item_equipment_extra
  DROP CONSTRAINT IF EXISTS chk_item_equipment_enhance_gold_fixed;
ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_enhance_gold_fixed CHECK (enhance_gold_increment_fixed >= 0);

ALTER TABLE item_equipment_extra
  DROP CONSTRAINT IF EXISTS chk_item_equipment_enhance_gold_percent;
ALTER TABLE item_equipment_extra
  ADD CONSTRAINT chk_item_equipment_enhance_gold_percent CHECK (enhance_gold_increment_percent >= 0 AND enhance_gold_increment_percent <= 1000);

COMMENT ON COLUMN item_equipment_extra.enhance_gold_cost_enabled IS '是否对该装备启用强化铜币消耗';
COMMENT ON COLUMN item_equipment_extra.enhance_gold_base_copper IS '强化至 +1 的基础铜币消耗';
COMMENT ON COLUMN item_equipment_extra.enhance_gold_increment_mode IS 'fixed=每级固定增加；percent=每级百分比复合递增';
COMMENT ON COLUMN item_equipment_extra.enhance_gold_increment_fixed IS 'fixed 模式下每升 1 级增加的铜币';
COMMENT ON COLUMN item_equipment_extra.enhance_gold_increment_percent IS 'percent 模式下每升 1 级的复合递增百分比';
