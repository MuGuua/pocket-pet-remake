-- 065_equipment_enhance_cost_gold_copper.sql
-- 装备强化按目标等级配置铜币消耗（总铜币真值），供客户端强化页展示与服务端扣费。

ALTER TABLE equipment_enhance_cost
  ADD COLUMN IF NOT EXISTS cost_gold_copper BIGINT NOT NULL DEFAULT 0;

ALTER TABLE equipment_enhance_cost
  DROP CONSTRAINT IF EXISTS chk_equipment_enhance_cost_gold_copper;

ALTER TABLE equipment_enhance_cost
  ADD CONSTRAINT chk_equipment_enhance_cost_gold_copper CHECK (cost_gold_copper >= 0);

COMMENT ON COLUMN equipment_enhance_cost.cost_gold_copper IS
  '强化至 target_level 所需铜币总量（currency_copper_total 最小单位），与 player_wallet 扣费口径一致';

UPDATE equipment_enhance_cost
SET cost_gold_copper = CASE
  WHEN target_level <= 5 THEN 500
  WHEN target_level <= 10 THEN 1000
  ELSE 2000
END,
updated_at = CURRENT_TIMESTAMP
WHERE cost_gold_copper = 0;
