-- 将其余仍为 notice 的 dialog 菜单项迁入结构化剧情表，便于统一走 dialogue 链路。

UPDATE npc_menu_entry
SET action_result_type = 'dialogue',
    action_notice = '',
    updated_at = CURRENT_TIMESTAMP
WHERE (entity_id = 93001 AND entry_id = 'dialog_market_news')
   OR (entity_id = 91001 AND entry_id = 'dialog_warehouse_intro')
   OR (entity_id = 93002 AND entry_id = 'dialog_trade_tip');

INSERT INTO npc_dialogue (
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status
) VALUES
  (93001, 'dialog_market_news', 'radiant_market_news', '市场新鲜事', 'start', 1, 1),
  (91001, 'dialog_warehouse_intro', 'warehouse_intro', '仓库介绍', 'start', 1, 1),
  (93002, 'dialog_trade_tip', 'market_trade_tip', '讨价还价', 'start', 1, 1)
ON CONFLICT (entity_id, entry_id) DO UPDATE SET
  dialogue_code = EXCLUDED.dialogue_code,
  title = EXCLUDED.title,
  start_node_id = EXCLUDED.start_node_id,
  version = EXCLUDED.version,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

WITH target_dialogue AS (
  SELECT dialogue_id, entity_id, entry_id
  FROM npc_dialogue
  WHERE (entity_id = 93001 AND entry_id = 'dialog_market_news')
     OR (entity_id = 91001 AND entry_id = 'dialog_warehouse_intro')
     OR (entity_id = 93002 AND entry_id = 'dialog_trade_tip')
)
INSERT INTO npc_dialogue_node (
  dialogue_id,
  node_id,
  node_type,
  speaker,
  content,
  content_format,
  portrait_key,
  next_node_id,
  client_animation_key,
  client_animation_block,
  conditions_json,
  effects_json,
  sort_order
)
SELECT d.dialogue_id, 'start', 'line',
  CASE d.entry_id
    WHEN 'dialog_market_news' THEN '市场理萌'
    WHEN 'dialog_warehouse_intro' THEN '罗思'
    WHEN 'dialog_trade_tip' THEN '罗格'
  END,
  CASE d.entry_id
    WHEN 'dialog_market_news' THEN '最近市场新开了几家铺子，有空记得多逛逛。'
    WHEN 'dialog_warehouse_intro' THEN '这里负责保管训练家暂时寄存的物资，有需要随时来找我。'
    WHEN 'dialog_trade_tip' THEN '买卖讲究货比三家，别急着出手。'
  END,
  'plain',
  CASE d.entry_id
    WHEN 'dialog_market_news' THEN 'npc_limeng_smile'
    WHEN 'dialog_warehouse_intro' THEN 'npc_luosi_normal'
    WHEN 'dialog_trade_tip' THEN 'npc_luoge_normal'
  END,
  'end', '', 0, '{}'::jsonb, '{}'::jsonb, 10
FROM target_dialogue d
UNION ALL
SELECT d.dialogue_id, 'end', 'end', '', '', 'plain', '', '', '', 0, '{}'::jsonb, '{}'::jsonb, 99
FROM target_dialogue d
ON CONFLICT (dialogue_id, node_id) DO UPDATE SET
  node_type = EXCLUDED.node_type,
  speaker = EXCLUDED.speaker,
  content = EXCLUDED.content,
  content_format = EXCLUDED.content_format,
  portrait_key = EXCLUDED.portrait_key,
  next_node_id = EXCLUDED.next_node_id,
  client_animation_key = EXCLUDED.client_animation_key,
  client_animation_block = EXCLUDED.client_animation_block,
  conditions_json = EXCLUDED.conditions_json,
  effects_json = EXCLUDED.effects_json,
  sort_order = EXCLUDED.sort_order;
