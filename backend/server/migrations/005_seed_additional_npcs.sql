INSERT INTO world_entity_definition (
  entity_id,
  entity_code,
  display_name,
  entity_type,
  scene_id,
  pos_x,
  pos_y,
  dir,
  speed,
  status
) VALUES
  (
    91010,
    'warehouse_keeper_aqing',
    '仓库管理员阿青',
    2,
    1,
    7,
    6,
    2,
    0,
    1
  ),
  (
    91020,
    'market_shopkeeper_bolin',
    '商店老板柏林',
    2,
    3,
    15,
    7,
    2,
    0,
    1
  )
ON CONFLICT (entity_id) DO UPDATE SET
  entity_code = EXCLUDED.entity_code,
  display_name = EXCLUDED.display_name,
  entity_type = EXCLUDED.entity_type,
  scene_id = EXCLUDED.scene_id,
  pos_x = EXCLUDED.pos_x,
  pos_y = EXCLUDED.pos_y,
  dir = EXCLUDED.dir,
  speed = EXCLUDED.speed,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

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
    91010,
    'dialog_intro',
    'dialog',
    '打个招呼',
    '先认识一下仓库管理员',
    'available',
    100,
    10,
    'notice',
    '阿青说：欢迎来到仓库，这里会帮你保管暂时寄存的物资。',
    1
  ),
  (
    91010,
    'dialog_tip',
    'dialog',
    '仓库说明',
    '看看这里以后能做什么',
    'available',
    80,
    20,
    'notice',
    '阿青说：后续这里会开放道具寄存、整理和批量领取功能。',
    1
  ),
  (
    91020,
    'shop_open_basic',
    'shop',
    '打开商店',
    '浏览基础商品',
    'available',
    100,
    10,
    'notice',
    '商店系统正在接入中，当前先返回占位提示。',
    1
  ),
  (
    91020,
    'dialog_discount',
    'dialog',
    '折扣消息',
    '听听老板最近的活动',
    'available',
    80,
    20,
    'notice',
    '柏林说：这周训练补给有折扣，记得多来看看。',
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
