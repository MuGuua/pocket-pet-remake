-- 主线《旅行的起点》1/5 至 3/5：任务模板、场景提示、NPC 菜单和结构化对话。
ALTER TABLE scene_entry_trigger
  ADD COLUMN IF NOT EXISTS prompt_text TEXT NOT NULL DEFAULT '';

-- 停用旧版演示主线，避免与新手主线同时出现在 NPC 和任务面板中。
UPDATE quest_template
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE UPPER(quest_type) = 'MAIN'
  AND quest_id NOT IN (1101, 1102, 1103);

INSERT INTO quest_template (
  quest_id, quest_type, name, title, description, chapter, sort_order,
  accept_mode, submit_mode, can_abandon, is_repeatable, auto_track,
  client_icon_id, start_npc_id, submit_npc_id, min_player_level,
  visible_conditions, unlock_conditions, accept_conditions, pre_quest_ids,
  objectives_json, rewards_json, tags_json, time_window_json,
  accept_animation_key, submit_animation_key, version, status
) VALUES
(
  1101, 'MAIN', 'main_journey_start_1', '主线·旅行的起点(1/5)',
  '听取桃子的旅行建议，然后前往市场拜访生产导师·璃梦。', 1, 10,
  'SYSTEM', 'AUTO', FALSE, FALSE, TRUE,
  1, 0, 0, 1,
  '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb,
  '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"与生产导师·璃梦对话","target":1,"target_selector":{"npc_id":93001},"guide":{"scene_id":3,"npc_id":93001,"text":"向左进入市场，找到生产导师·璃梦"}}]'::jsonb,
  '[{"type":"gold","value":100}]'::jsonb,
  '["main","newbie","journey_start"]'::jsonb, '{}'::jsonb, '', '', 1, 1
),
(
  1102, 'MAIN', 'main_journey_start_2', '主线·旅行的起点(2/5)',
  '继续听生产导师·璃梦讲解在闪光镇旅行和生产的基础知识。', 1, 20,
  'SYSTEM', 'AUTO', FALSE, FALSE, TRUE,
  1, 0, 0, 1,
  '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[1101]'::jsonb,
  '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"继续与生产导师·璃梦对话","target":1,"target_selector":{"npc_id":93001},"guide":{"scene_id":3,"npc_id":93001,"text":"继续听生产导师·璃梦的介绍"}}]'::jsonb,
  '[{"type":"gold","value":150}]'::jsonb,
  '["main","newbie","journey_start"]'::jsonb, '{}'::jsonb, '', '', 1, 1
),
(
  1103, 'MAIN', 'main_journey_start_3', '主线·旅行的起点(3/5)',
  '拜访同一市场中的杂货商人·罗格，了解旅行物资的准备方法。', 1, 30,
  'SYSTEM', 'AUTO', FALSE, FALSE, TRUE,
  1, 0, 0, 1,
  '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[1102]'::jsonb,
  '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"与杂货商人·罗格对话","target":1,"target_selector":{"npc_id":93002},"guide":{"scene_id":3,"npc_id":93002,"text":"在市场找到杂货商人·罗格"}}]'::jsonb,
  '[{"type":"gold","value":200}]'::jsonb,
  '["main","newbie","journey_start"]'::jsonb, '{}'::jsonb, '', '', 1, 1
)
ON CONFLICT (quest_id) DO UPDATE SET
  quest_type = EXCLUDED.quest_type, name = EXCLUDED.name, title = EXCLUDED.title,
  description = EXCLUDED.description, chapter = EXCLUDED.chapter, sort_order = EXCLUDED.sort_order,
  accept_mode = EXCLUDED.accept_mode, submit_mode = EXCLUDED.submit_mode,
  can_abandon = EXCLUDED.can_abandon, is_repeatable = EXCLUDED.is_repeatable,
  auto_track = EXCLUDED.auto_track, client_icon_id = EXCLUDED.client_icon_id,
  start_npc_id = EXCLUDED.start_npc_id, submit_npc_id = EXCLUDED.submit_npc_id,
  min_player_level = EXCLUDED.min_player_level, visible_conditions = EXCLUDED.visible_conditions,
  unlock_conditions = EXCLUDED.unlock_conditions, accept_conditions = EXCLUDED.accept_conditions,
  pre_quest_ids = EXCLUDED.pre_quest_ids, objectives_json = EXCLUDED.objectives_json,
  rewards_json = EXCLUDED.rewards_json, tags_json = EXCLUDED.tags_json,
  time_window_json = EXCLUDED.time_window_json,
  accept_animation_key = EXCLUDED.accept_animation_key,
  submit_animation_key = EXCLUDED.submit_animation_key,
  version = EXCLUDED.version, status = EXCLUDED.status, updated_at = CURRENT_TIMESTAMP;

-- 初见桃子动画结束后先显示提示，确认后再解锁桃子；即使清理过旧主线也能重新创建。
INSERT INTO scene_entry_trigger (
  scene_id, trigger_code, priority, once_flag_key,
  required_quest_id, required_quest_state, forbidden_quest_id, forbidden_quest_state,
  client_animation_key, prompt_text, block_movement,
  effect_accept_quest_id, effect_set_flags, status
) VALUES (
  2, 'first_enter_east_road_taozi', 100, 'intro_taozi_played',
  0, '', 0, '',
  '初见桃子', '剧情结束了。和桃子谈谈，开始你的第一次旅行吧。', TRUE,
  0, '["taozi_npc_unlocked"]'::jsonb, 1
)
ON CONFLICT (trigger_code) DO UPDATE SET
  scene_id = EXCLUDED.scene_id, priority = EXCLUDED.priority,
  once_flag_key = EXCLUDED.once_flag_key,
  required_quest_id = EXCLUDED.required_quest_id,
  required_quest_state = EXCLUDED.required_quest_state,
  forbidden_quest_id = EXCLUDED.forbidden_quest_id,
  forbidden_quest_state = EXCLUDED.forbidden_quest_state,
  client_animation_key = EXCLUDED.client_animation_key,
  prompt_text = EXCLUDED.prompt_text, block_movement = EXCLUDED.block_movement,
  effect_accept_quest_id = EXCLUDED.effect_accept_quest_id,
  effect_set_flags = EXCLUDED.effect_set_flags, status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- 首次进入市场只显示一次提示面板，不播放额外剧情。
INSERT INTO scene_entry_trigger (
  scene_id, trigger_code, priority, once_flag_key,
  client_animation_key, prompt_text, block_movement,
  effect_accept_quest_id, effect_set_flags, status
) VALUES (
  3, 'first_enter_market_journey_start', 90, 'market_first_visit_prompted',
  '', '这里是闪光镇市场。找到生产导师·璃梦，继续《主线·旅行的起点(1/5)》。', TRUE,
  0, '[]'::jsonb, 1
)
ON CONFLICT (trigger_code) DO UPDATE SET
  scene_id = EXCLUDED.scene_id, priority = EXCLUDED.priority,
  once_flag_key = EXCLUDED.once_flag_key, client_animation_key = EXCLUDED.client_animation_key,
  prompt_text = EXCLUDED.prompt_text, block_movement = EXCLUDED.block_movement,
  effect_accept_quest_id = EXCLUDED.effect_accept_quest_id,
  effect_set_flags = EXCLUDED.effect_set_flags, status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

UPDATE world_entity_definition
SET display_name = CASE entity_id
      WHEN 93001 THEN '生产导师·璃梦'
      WHEN 93002 THEN '杂货商人·罗格'
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE entity_id IN (93001, 93002);

UPDATE world_entity_definition
SET scene_id = 2,
    status = 1,
    updated_at = CURRENT_TIMESTAMP
WHERE entity_id = 92001;

-- 四个任务菜单均使用 dialogue 类型，避免客户端绕过“接受/取消”对话直接接取。
INSERT INTO npc_menu_entry (
  entity_id, entry_id, entry_type, title, subtitle, state,
  priority, sort_order, action_result_type, action_notice,
  conditions_json, linked_quest_id, status
) VALUES
  (92001, 'main_journey_1_accept', 'dialogue', '主线·旅行的起点(1/5)', '', 'available', 200, 1, 'dialogue', '', '{"quest_id":1101,"quest_state":"AVAILABLE"}'::jsonb, 1101, 1),
  (93001, 'main_journey_1_continue', 'dialogue', '主线·旅行的起点(1/5)', '', 'available', 200, 1, 'dialogue', '', '{"quest_id":1101,"quest_state":"ACCEPTED"}'::jsonb, 1101, 1),
  (93001, 'main_journey_2_accept', 'dialogue', '主线·旅行的起点(2/5)', '', 'available', 190, 2, 'dialogue', '', '{"quest_id":1102,"quest_state":"AVAILABLE"}'::jsonb, 1102, 1),
  (93002, 'main_journey_3_accept', 'dialogue', '主线·旅行的起点(3/5)', '', 'available', 200, 1, 'dialogue', '', '{"quest_id":1103,"quest_state":"AVAILABLE"}'::jsonb, 1103, 1)
ON CONFLICT (entity_id, entry_id) DO UPDATE SET
  entry_type = EXCLUDED.entry_type, title = EXCLUDED.title, subtitle = EXCLUDED.subtitle,
  state = EXCLUDED.state, priority = EXCLUDED.priority, sort_order = EXCLUDED.sort_order,
  action_result_type = EXCLUDED.action_result_type, action_notice = EXCLUDED.action_notice,
  conditions_json = EXCLUDED.conditions_json, linked_quest_id = EXCLUDED.linked_quest_id,
  status = EXCLUDED.status, updated_at = CURRENT_TIMESTAMP;

INSERT INTO npc_dialogue (entity_id, entry_id, dialogue_code, title, start_node_id, version, status) VALUES
  (92001, 'main_journey_1_accept', 'main_journey_1_accept', '旅行的起点 1/5', 'start', 1, 1),
  (93001, 'main_journey_1_continue', 'main_journey_1_continue', '旅行的起点 1/5', 'start', 1, 1),
  (93001, 'main_journey_2_accept', 'main_journey_2_accept', '旅行的起点 2/5', 'start', 1, 1),
  (93002, 'main_journey_3_accept', 'main_journey_3_accept', '旅行的起点 3/5', 'start', 1, 1)
ON CONFLICT (entity_id, entry_id) DO UPDATE SET
  dialogue_code = EXCLUDED.dialogue_code, title = EXCLUDED.title,
  start_node_id = EXCLUDED.start_node_id, version = EXCLUDED.version,
  status = EXCLUDED.status, updated_at = CURRENT_TIMESTAMP;

-- 重建这四段主线对话的节点与选项，保证迁移重复执行时不会保留旧分支。
DELETE FROM npc_dialogue_node
WHERE dialogue_id IN (
  SELECT dialogue_id FROM npc_dialogue
  WHERE (entity_id, entry_id) IN (
    (92001, 'main_journey_1_accept'),
    (93001, 'main_journey_1_continue'),
    (93001, 'main_journey_2_accept'),
    (93002, 'main_journey_3_accept')
  )
);

WITH dialogues AS (
  SELECT dialogue_id, entry_id FROM npc_dialogue
  WHERE (entity_id, entry_id) IN (
    (92001, 'main_journey_1_accept'),
    (93001, 'main_journey_1_continue'),
    (93001, 'main_journey_2_accept'),
    (93002, 'main_journey_3_accept')
  )
)
INSERT INTO npc_dialogue_node (
  dialogue_id, node_id, node_type, speaker, content, content_format,
  portrait_key, next_node_id, client_animation_key, client_animation_block,
  conditions_json, effects_json, sort_order
)
SELECT dialogue_id, 'start', 'choice', '桃子',
  '我要去市场准备旅行物资，你也一起熟悉一下闪光镇吧。愿意现在出发吗？', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 10 FROM dialogues WHERE entry_id = 'main_journey_1_accept'
UNION ALL
SELECT dialogue_id, 'accepted', 'line', '桃子',
  '太好了！先向左走进市场，找到生产导师·璃梦，她会告诉你旅行前要准备什么。', 'plain', '', 'briefing', '', 0,
  '{}'::jsonb, '{"accept_quest_id":1101}'::jsonb, 20 FROM dialogues WHERE entry_id = 'main_journey_1_accept'
UNION ALL
SELECT dialogue_id, 'briefing', 'line', '桃子',
  '沿着东路一直向左就到了。别担心，我会在这里等你的好消息。', 'plain', '', 'accepted_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 30 FROM dialogues WHERE entry_id = 'main_journey_1_accept'
UNION ALL
SELECT dialogue_id, 'cancel', 'line', '桃子', '没关系，准备好了再来找我。', 'plain', '', 'cancel_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 40 FROM dialogues WHERE entry_id = 'main_journey_1_accept'
UNION ALL
SELECT dialogue_id, 'accepted_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{"notice":"已接受《主线·旅行的起点(1/5)》，向左进入市场。"}'::jsonb, 98 FROM dialogues WHERE entry_id = 'main_journey_1_accept'
UNION ALL
SELECT dialogue_id, 'cancel_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 99 FROM dialogues WHERE entry_id = 'main_journey_1_accept'

UNION ALL
SELECT dialogue_id, 'start', 'line', '生产导师·璃梦',
  '桃子让你来找我，对吧？旅行并不只是向前走，也要学会准备和补给。', 'plain', '', 'finish', '', 0,
  '{}'::jsonb, '{}'::jsonb, 10 FROM dialogues WHERE entry_id = 'main_journey_1_continue'
UNION ALL
SELECT dialogue_id, 'finish', 'line', '生产导师·璃梦',
  '先记住这一点：每次离开城镇前，都要检查行囊和伙伴的状态。', 'plain', '', 'end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 20 FROM dialogues WHERE entry_id = 'main_journey_1_continue'
UNION ALL
SELECT dialogue_id, 'end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{"quest_event":"TALK_TO_NPC","submit_quest_id":1101,"notice":"《主线·旅行的起点(1/5)》完成，奖励已发放。"}'::jsonb, 99 FROM dialogues WHERE entry_id = 'main_journey_1_continue'

UNION ALL
SELECT dialogue_id, 'start', 'choice', '生产导师·璃梦',
  '接下来我可以教你辨认基础生产材料。这会让之后的旅行轻松很多，要现在开始吗？', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 10 FROM dialogues WHERE entry_id = 'main_journey_2_accept'
UNION ALL
SELECT dialogue_id, 'accepted', 'line', '生产导师·璃梦',
  '很好。材料最重要的不是稀有，而是能不能在需要的时候派上用场。', 'plain', '', 'finish', '', 0,
  '{}'::jsonb, '{"accept_quest_id":1102}'::jsonb, 20 FROM dialogues WHERE entry_id = 'main_journey_2_accept'
UNION ALL
SELECT dialogue_id, 'finish', 'line', '生产导师·璃梦',
  '今天先学到这里。市场里的杂货商人·罗格更懂旅行补给，你可以去找他。', 'plain', '', 'accepted_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 30 FROM dialogues WHERE entry_id = 'main_journey_2_accept'
UNION ALL
SELECT dialogue_id, 'cancel', 'line', '生产导师·璃梦', '没问题，想学的时候再来找我。', 'plain', '', 'cancel_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 40 FROM dialogues WHERE entry_id = 'main_journey_2_accept'
UNION ALL
SELECT dialogue_id, 'accepted_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{"quest_event":"TALK_TO_NPC","submit_quest_id":1102,"notice":"《主线·旅行的起点(2/5)》完成，奖励已发放。"}'::jsonb, 98 FROM dialogues WHERE entry_id = 'main_journey_2_accept'
UNION ALL
SELECT dialogue_id, 'cancel_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 99 FROM dialogues WHERE entry_id = 'main_journey_2_accept'

UNION ALL
SELECT dialogue_id, 'start', 'choice', '杂货商人·罗格',
  '璃梦说你需要学习旅行补给？我可以从最实用的行囊清单讲起，要听吗？', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 10 FROM dialogues WHERE entry_id = 'main_journey_3_accept'
UNION ALL
SELECT dialogue_id, 'accepted', 'line', '杂货商人·罗格',
  '先准备恢复用品，再考虑其他东西。能平安回来，才有下一次旅行。', 'plain', '', 'finish', '', 0,
  '{}'::jsonb, '{"accept_quest_id":1103}'::jsonb, 20 FROM dialogues WHERE entry_id = 'main_journey_3_accept'
UNION ALL
SELECT dialogue_id, 'finish', 'line', '杂货商人·罗格',
  '记住这份清单。等你真正出发时，它会帮上大忙。', 'plain', '', 'accepted_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 30 FROM dialogues WHERE entry_id = 'main_journey_3_accept'
UNION ALL
SELECT dialogue_id, 'cancel', 'line', '杂货商人·罗格', '不着急，想清楚了再来。', 'plain', '', 'cancel_end', '', 0,
  '{}'::jsonb, '{}'::jsonb, 40 FROM dialogues WHERE entry_id = 'main_journey_3_accept'
UNION ALL
SELECT dialogue_id, 'accepted_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{"quest_event":"TALK_TO_NPC","submit_quest_id":1103,"notice":"《主线·旅行的起点(3/5)》完成，奖励已发放。"}'::jsonb, 98 FROM dialogues WHERE entry_id = 'main_journey_3_accept'
UNION ALL
SELECT dialogue_id, 'cancel_end', 'end', '', '', 'plain', '', '', '', 0,
  '{}'::jsonb, '{}'::jsonb, 99 FROM dialogues WHERE entry_id = 'main_journey_3_accept';

WITH dialogues AS (
  SELECT dialogue_id, entry_id FROM npc_dialogue
  WHERE entry_id IN ('main_journey_1_accept', 'main_journey_2_accept', 'main_journey_3_accept')
)
INSERT INTO npc_dialogue_option (
  dialogue_id, node_id, option_id, option_text, option_format,
  next_node_id, conditions_json, effects_json, sort_order
)
SELECT dialogue_id, 'start', 'accept', '接受', 'plain', 'accepted', '{}'::jsonb, '{}'::jsonb, 10 FROM dialogues
UNION ALL
SELECT dialogue_id, 'start', 'cancel', '取消', 'plain', 'cancel', '{}'::jsonb, '{}'::jsonb, 20 FROM dialogues
ON CONFLICT (dialogue_id, node_id, option_id) DO UPDATE SET
  option_text = EXCLUDED.option_text, option_format = EXCLUDED.option_format,
  next_node_id = EXCLUDED.next_node_id, conditions_json = EXCLUDED.conditions_json,
  effects_json = EXCLUDED.effects_json, sort_order = EXCLUDED.sort_order;
