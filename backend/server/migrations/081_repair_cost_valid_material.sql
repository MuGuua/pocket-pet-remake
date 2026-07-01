-- 081_repair_cost_valid_material.sql
-- 修复装备修复材料配置：只允许启用 item_sub_type=equipment_repair 的材料作为运行时修复消耗。
--
-- 背景：
-- 运行时修复接口会读取 equipment_repair_cost 中 status=1 的配置。
-- 如果历史数据里存在非修复材料且 item_id 更小的启用记录，服务端会误取该记录并报
-- `invalid repair material item`，导致玩家即使有修复宝石也无法修复损坏装备。

-- 确保默认修复宝石模板存在且可用。正式玩法数据仍以数据库为事实来源，客户端只展示服务端快照。
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

-- 禁用所有指向非修复材料或已禁用道具的修复消耗配置，避免运行时误选脏数据。
UPDATE equipment_repair_cost erc
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE erc.status = 1
  AND NOT EXISTS (
    SELECT 1
    FROM item_definition idf
    WHERE idf.item_id = erc.cost_item_id
      AND idf.is_enabled = TRUE
      AND LOWER(TRIM(idf.item_sub_type)) = 'equipment_repair'
  );

-- 确保默认修复消耗配置可用；如果线上没有自定义修复材料，运行时会稳定使用该配置。
INSERT INTO equipment_repair_cost (cost_item_id, cost_quantity, description, status)
VALUES (3202, 1, '修复损坏装备默认消耗', 1)
ON CONFLICT (cost_item_id) DO UPDATE SET
  cost_quantity = EXCLUDED.cost_quantity,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
