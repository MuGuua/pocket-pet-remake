-- NPC 菜单项增加任务可见性条件与任务绑定字段，支持按任务阶段控制菜单显示。

ALTER TABLE npc_menu_entry
  ADD COLUMN IF NOT EXISTS conditions_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS linked_quest_id BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN npc_menu_entry.conditions_json IS
  '菜单可见性条件 JSON：quest_id、quest_state、objective_id、objective_completed';
COMMENT ON COLUMN npc_menu_entry.linked_quest_id IS
  'entry_type=quest 且 action_result_type=quest_accept/quest_submit 时绑定的任务模板 ID';
