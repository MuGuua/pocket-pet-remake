-- 015_seed_warehouse_menu_entries.sql
--
-- 为仓库 NPC 补齐正式的“打开仓库”菜单项。
-- 客户端只消费服务端从 npc_menu_entry 返回的 entry_type，不在代码里硬编码正式 NPC 功能入口。

INSERT INTO npc_menu_entry (
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  sort_order,
  action_result_type,
  action_notice,
  status
) VALUES
  (
    91001,
    'warehouse_open_primary',
    'warehouse',
    '打开仓库',
    '查看寄存中的道具与装备',
    'available',
    120,
    5,
    'panel',
    '',
    1
  ),
  (
    91010,
    'warehouse_open_primary',
    'warehouse',
    '打开仓库',
    '查看寄存中的道具与装备',
    'available',
    120,
    5,
    'panel',
    '',
    1
  )
ON CONFLICT (entity_id, entry_id) DO UPDATE SET
  entry_type = EXCLUDED.entry_type,
  title = EXCLUDED.title,
  subtitle = EXCLUDED.subtitle,
  state = EXCLUDED.state,
  priority = EXCLUDED.priority,
  sort_order = EXCLUDED.sort_order,
  action_result_type = EXCLUDED.action_result_type,
  action_notice = EXCLUDED.action_notice,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
