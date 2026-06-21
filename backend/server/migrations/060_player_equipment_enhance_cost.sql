-- 060_player_equipment_enhance_cost.sql
-- 人物装备强化材料消耗配置与测试用强化石道具。

CREATE TABLE IF NOT EXISTS equipment_enhance_cost (
  target_level INTEGER PRIMARY KEY,
  cost_item_id BIGINT NOT NULL REFERENCES item_definition(item_id),
  cost_quantity BIGINT NOT NULL DEFAULT 1,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_enhance_cost_quantity CHECK (cost_quantity > 0),
  CONSTRAINT chk_equipment_enhance_cost_target_level CHECK (target_level >= 1 AND target_level <= 15)
);

DROP TRIGGER IF EXISTS trg_equipment_enhance_cost_updated_at ON equipment_enhance_cost;
CREATE TRIGGER trg_equipment_enhance_cost_updated_at
BEFORE UPDATE ON equipment_enhance_cost
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO item_definition (
  item_id, item_code, item_name, item_type, item_sub_type, quality, rarity, icon, "desc",
  max_stack, occupy_slots, auto_merge, sort_weight, usable, use_scope, target_type,
  required_level, required_scene_id, bind_type, can_sell, can_drop, can_store, can_trade,
  expire_at_rule, effect_type, effect_value, effect_params_json,
  buy_price_copper, sell_price_copper, recycle_price_copper, price_type, is_enabled
) VALUES (
  3201, 'equipment_enhance_stone', '强化石', 'material', 'equipment_enhance', 2, 1, '',
  '用于人物装备强化的基础材料。',
  999, 1, TRUE, 200, FALSE, 'none', 'none',
  1, 0, 'none', TRUE, FALSE, TRUE, TRUE,
  '', '', 0, '{}'::jsonb,
  500, 100, 50, 'base_coin', TRUE
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

INSERT INTO equipment_enhance_cost (target_level, cost_item_id, cost_quantity, description, status)
SELECT
  level,
  3201,
  CASE
    WHEN level <= 5 THEN 1
    WHEN level <= 10 THEN 2
    ELSE 3
  END,
  '强化至 +' || level::text,
  1
FROM generate_series(1, 15) AS level
ON CONFLICT (target_level) DO UPDATE SET
  cost_item_id = EXCLUDED.cost_item_id,
  cost_quantity = EXCLUDED.cost_quantity,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
