-- NPC 商店商品表：把每个商店 NPC 的可售物品配置持久化到数据库，客户端只消费服务端返回的商品列表。

CREATE TABLE IF NOT EXISTS npc_shop_goods (
  entity_id BIGINT NOT NULL REFERENCES world_entity_definition(entity_id) ON DELETE CASCADE,
  item_id BIGINT NOT NULL REFERENCES item_definition(item_id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (entity_id, item_id)
);

UPDATE npc_menu_entry
SET action_result_type = 'shop',
    action_notice = '',
    updated_at = CURRENT_TIMESTAMP
WHERE entity_id = 93002
  AND entry_id = 'shop_open_market';

INSERT INTO npc_shop_goods (entity_id, item_id, sort_order, status)
VALUES
  (93002, 3003, 10, 1),
  (93002, 3004, 20, 1)
ON CONFLICT (entity_id, item_id) DO UPDATE SET
  sort_order = EXCLUDED.sort_order,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
