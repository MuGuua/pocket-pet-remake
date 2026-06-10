CREATE TABLE IF NOT EXISTS world_entity_definition (
  entity_id BIGINT PRIMARY KEY,
  entity_code TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  entity_type INTEGER NOT NULL DEFAULT 2,
  scene_id INTEGER NOT NULL,
  pos_x INTEGER NOT NULL,
  pos_y INTEGER NOT NULL,
  dir INTEGER NOT NULL DEFAULT 2,
  speed INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS npc_menu_entry (
  entity_id BIGINT NOT NULL REFERENCES world_entity_definition(entity_id) ON DELETE CASCADE,
  entry_id TEXT NOT NULL,
  entry_type TEXT NOT NULL,
  title TEXT NOT NULL,
  subtitle TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'available',
  priority INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  action_result_type TEXT NOT NULL DEFAULT 'notice',
  action_notice TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (entity_id, entry_id)
);

CREATE INDEX IF NOT EXISTS idx_world_entity_definition_scene_status
  ON world_entity_definition (scene_id, status, entity_id);

CREATE INDEX IF NOT EXISTS idx_npc_menu_entry_entity_status
  ON npc_menu_entry (entity_id, status, priority DESC, sort_order ASC);

INSERT INTO world_entity_definition (
  entity_id, entity_code, display_name, entity_type, scene_id, pos_x, pos_y, dir, speed, status
) VALUES
  (90001, 'guide_npc', 'GuideNPC', 2, 1, 10, 6, 2, 0, 1),
  (91001, 'warehouse_luosi', '罗思', 2, 1, 6, 6, 2, 0, 1),
  (90002, 'station_keeper', 'StationKeeper', 2, 2, 2, 3, 1, 0, 1),
  (93001, 'radiant_market_limeng', '市场理萌', 2, 3, 13, 8, 2, 0, 1),
  (93002, 'radiant_market_luoge', '市场罗格', 2, 3, 14, 6, 2, 0, 1),
  (90004, 'north_field_scout', 'NorthFieldScout', 2, 4, 4, 7, 2, 0, 1),
  (90005, 'school_caretaker', 'SchoolCaretaker', 2, 5, 9, 4, 1, 0, 1),
  (90006, 'battle_guide', 'BattleGuide', 2, 6, 7, 8, 0, 0, 1)
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
  entity_id, entry_id, entry_type, title, subtitle, state, priority, sort_order, action_result_type, action_notice, status
) VALUES
  (
    91001, 'dialog_warehouse_intro', 'dialog',
    '仓库介绍', '问问仓库平时负责什么', 'available', 80, 10, 'notice',
    '罗思说：这里负责保管训练家暂时寄存的物资。', 1
  ),
  (
    93001, 'dialog_market_news', 'dialog',
    '打听消息', '问问市场最近的新鲜事', 'available', 80, 10, 'notice',
    '理萌说：最近市场新开了几家铺子。', 1
  ),
  (
    93002, 'shop_open_market', 'shop',
    '打开商店', '浏览基础商品（占位）', 'available', 100, 10, 'notice',
    '商店面板待接入，当前先返回占位提示。', 1
  ),
  (
    93002, 'dialog_trade_tip', 'dialog',
    '讨价还价', '听听老商贩的经验', 'available', 70, 20, 'notice',
    '罗格说：买卖讲究货比三家。', 1
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
