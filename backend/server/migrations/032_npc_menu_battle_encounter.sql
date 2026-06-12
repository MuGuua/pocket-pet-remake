-- 032_npc_menu_battle_encounter.sql
-- NPC 菜单「挑战」项：绑定 monster_encounter.entity_id，玩家点击后直接触发 PVE 固定战。

ALTER TABLE npc_menu_entry
  ADD COLUMN IF NOT EXISTS battle_encounter_entity_id BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN npc_menu_entry.battle_encounter_entity_id IS
  'entry_type/action_result_type 为 battle 时绑定的 monster_encounter.entity_id；0 表示使用当前 NPC entity_id';

-- 战斗引导 NPC 示例菜单：挑战 -> 90006 固定战遭遇
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
  battle_encounter_entity_id,
  status
) VALUES (
  90006,
  'battle_guide_challenge',
  'battle',
  '挑战',
  '与训练师进行友好对决',
  'available',
  110,
  5,
  'battle',
  '',
  90006,
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
  battle_encounter_entity_id = EXCLUDED.battle_encounter_entity_id,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
