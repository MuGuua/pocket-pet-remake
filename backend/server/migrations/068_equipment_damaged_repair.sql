-- 068_equipment_damaged_repair.sql
-- 强化失败损坏标记、修复宝石道具与修复消耗配置。

ALTER TABLE equipment_instance
ADD COLUMN IF NOT EXISTS is_damaged BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS equipment_repair_cost (
  cost_item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id),
  cost_quantity BIGINT NOT NULL DEFAULT 1,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_repair_cost_quantity CHECK (cost_quantity > 0)
);

DROP TRIGGER IF EXISTS trg_equipment_repair_cost_updated_at ON equipment_repair_cost;
CREATE TRIGGER trg_equipment_repair_cost_updated_at
BEFORE UPDATE ON equipment_repair_cost
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO item_definition (
  item_id, item_code, item_name, item_type, item_sub_type, quality, rarity, icon, "desc",
  max_stack, occupy_slots, auto_merge, sort_weight, usable, use_scope, target_type,
  required_level, required_scene_id, bind_type, can_sell, can_drop, can_store, can_trade,
  expire_at_rule, effect_type, effect_value, effect_params_json,
  buy_price_copper, sell_price_copper, recycle_price_copper, price_type, is_enabled
) VALUES (
  3202, 'equipment_repair_gem', '修复宝石', 'material', 'equipment_repair', 2, 1, '',
  '用于修复强化失败而损坏的人物装备。',
  999, 1, TRUE, 199, FALSE, 'none', 'none',
  1, 0, 'none', TRUE, FALSE, TRUE, TRUE,
  '', '', 0, '{}'::jsonb,
  800, 160, 80, 'base_coin', TRUE
)
ON CONFLICT (item_id) DO UPDATE SET
  item_code = EXCLUDED.item_code,
  item_name = EXCLUDED.item_name,
  item_type = EXCLUDED.item_type,
  item_sub_type = EXCLUDED.item_sub_type,
  "desc" = EXCLUDED."desc",
  max_stack = EXCLUDED.max_stack,
  is_enabled = EXCLUDED.is_enabled,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO equipment_repair_cost (cost_item_id, cost_quantity, description, status)
VALUES (3202, 1, '修复损坏装备默认消耗', 1)
ON CONFLICT (cost_item_id) DO UPDATE SET
  cost_quantity = EXCLUDED.cost_quantity,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
