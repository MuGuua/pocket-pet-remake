CREATE TABLE IF NOT EXISTS npc_dialogue (
  dialogue_id BIGSERIAL PRIMARY KEY,
  entity_id BIGINT NOT NULL REFERENCES world_entity_definition(entity_id) ON DELETE CASCADE,
  entry_id TEXT NOT NULL,
  dialogue_code TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  start_node_id TEXT NOT NULL DEFAULT 'start',
  version INTEGER NOT NULL DEFAULT 1,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (entity_id, entry_id)
);

CREATE TABLE IF NOT EXISTS npc_dialogue_node (
  dialogue_id BIGINT NOT NULL REFERENCES npc_dialogue(dialogue_id) ON DELETE CASCADE,
  node_id TEXT NOT NULL,
  node_type TEXT NOT NULL,
  speaker TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  content_format TEXT NOT NULL DEFAULT 'plain',
  portrait_key TEXT NOT NULL DEFAULT '',
  next_node_id TEXT NOT NULL DEFAULT '',
  client_animation_key TEXT NOT NULL DEFAULT '',
  client_animation_block SMALLINT NOT NULL DEFAULT 0,
  conditions_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  effects_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  sort_order INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (dialogue_id, node_id)
);

CREATE TABLE IF NOT EXISTS npc_dialogue_option (
  dialogue_id BIGINT NOT NULL,
  node_id TEXT NOT NULL,
  option_id TEXT NOT NULL,
  option_text TEXT NOT NULL,
  option_format TEXT NOT NULL DEFAULT 'plain',
  next_node_id TEXT NOT NULL DEFAULT '',
  conditions_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  effects_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  sort_order INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (dialogue_id, node_id, option_id),
  FOREIGN KEY (dialogue_id, node_id)
    REFERENCES npc_dialogue_node(dialogue_id, node_id)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS player_npc_dialogue_session (
  player_id BIGINT PRIMARY KEY,
  entity_id BIGINT NOT NULL,
  dialogue_id BIGINT NOT NULL REFERENCES npc_dialogue(dialogue_id) ON DELETE CASCADE,
  current_node_id TEXT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

UPDATE npc_menu_entry
SET action_result_type = 'dialogue',
    action_notice = '',
    updated_at = CURRENT_TIMESTAMP
WHERE entity_id = 93001
  AND entry_id = 'dialog_market_intro';

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
) VALUES (
  93001,
  'dialog_market_intro',
  'dialog',
  '让个路',
  '看看市场理萌的轻剧情演出',
  'available',
  90,
  5,
  'dialogue',
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

INSERT INTO npc_dialogue (
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status
) VALUES (
  93001,
  'dialog_market_intro',
  'radiant_market_intro',
  '市场理萌开场',
  'start',
  1,
  1
)
ON CONFLICT (entity_id, entry_id) DO UPDATE SET
  dialogue_code = EXCLUDED.dialogue_code,
  title = EXCLUDED.title,
  start_node_id = EXCLUDED.start_node_id,
  version = EXCLUDED.version,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

WITH target_dialogue AS (
  SELECT dialogue_id
  FROM npc_dialogue
  WHERE entity_id = 93001 AND entry_id = 'dialog_market_intro'
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
  sort_order
)
SELECT dialogue_id, 'start', 'line', '市场理萌', '你先稍等一下，我把前面的货箱挪开。', 'plain', 'npc_limeng_normal', 'move_aside', '', 0, 10
FROM target_dialogue
UNION ALL
SELECT dialogue_id, 'move_aside', 'action', '', '', 'plain', '', 'after_move', 'market_limeng_step_aside', 1, 20
FROM target_dialogue
UNION ALL
SELECT dialogue_id, 'after_move', 'choice', '市场理萌', '好啦，现在你是想先听听新鲜事，还是先逛逛？', 'plain', 'npc_limeng_smile', '', '', 0, 30
FROM target_dialogue
UNION ALL
SELECT dialogue_id, 'news', 'line', '市场理萌', '最近市场来了不少新货，记得多看看。', 'plain', 'npc_limeng_smile', 'end', '', 0, 40
FROM target_dialogue
UNION ALL
SELECT dialogue_id, 'end', 'end', '', '', 'plain', '', '', '', 0, 99
FROM target_dialogue
ON CONFLICT (dialogue_id, node_id) DO UPDATE SET
  node_type = EXCLUDED.node_type,
  speaker = EXCLUDED.speaker,
  content = EXCLUDED.content,
  content_format = EXCLUDED.content_format,
  portrait_key = EXCLUDED.portrait_key,
  next_node_id = EXCLUDED.next_node_id,
  client_animation_key = EXCLUDED.client_animation_key,
  client_animation_block = EXCLUDED.client_animation_block,
  sort_order = EXCLUDED.sort_order;

WITH target_dialogue AS (
  SELECT dialogue_id
  FROM npc_dialogue
  WHERE entity_id = 93001 AND entry_id = 'dialog_market_intro'
)
INSERT INTO npc_dialogue_option (
  dialogue_id,
  node_id,
  option_id,
  option_text,
  option_format,
  next_node_id,
  sort_order
)
SELECT dialogue_id, 'after_move', 'news', '听听新鲜事', 'plain', 'news', 10
FROM target_dialogue
UNION ALL
SELECT dialogue_id, 'after_move', 'leave', '先逛逛', 'plain', 'end', 20
FROM target_dialogue
ON CONFLICT (dialogue_id, node_id, option_id) DO UPDATE SET
  option_text = EXCLUDED.option_text,
  option_format = EXCLUDED.option_format,
  next_node_id = EXCLUDED.next_node_id,
  sort_order = EXCLUDED.sort_order;
