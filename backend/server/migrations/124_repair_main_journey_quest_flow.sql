BEGIN;

-- 修复《旅行的起点》任务模板被后台空数组覆盖后的配置漂移。
-- 只恢复空目标和空奖励，保留运营已填写的非空配置、完成提示与其他模板字段。
UPDATE quest_template
SET objectives_json = CASE
        WHEN objectives_json = '[]'::jsonb THEN
            CASE quest_id
                WHEN 1101 THEN '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"与生产导师·璃梦对话","target":1,"target_selector":{"npc_id":93001},"guide":{"scene_id":3,"npc_id":93001,"text":"向左进入市场，找到生产导师·璃梦"}}]'::jsonb
                WHEN 1102 THEN '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"继续与生产导师·璃梦对话","target":1,"target_selector":{"npc_id":93001},"guide":{"scene_id":3,"npc_id":93001,"text":"继续听生产导师·璃梦的介绍"}}]'::jsonb
                WHEN 1103 THEN '[{"objective_id":1,"event_type":"TALK_TO_NPC","description":"与杂货商人·罗格对话","target":1,"target_selector":{"npc_id":93002},"guide":{"scene_id":3,"npc_id":93002,"text":"在市场找到杂货商人·罗格"}}]'::jsonb
            END
        ELSE objectives_json
    END,
    rewards_json = CASE
        WHEN rewards_json = '[]'::jsonb THEN
            CASE quest_id
                WHEN 1101 THEN '[{"type":"gold","value":100}]'::jsonb
                WHEN 1102 THEN '[{"type":"gold","value":150}]'::jsonb
                WHEN 1103 THEN '[{"type":"gold","value":200}]'::jsonb
            END
        ELSE rewards_json
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE quest_id IN (1101, 1102, 1103)
  AND (
      objectives_json = '[]'::jsonb
      OR rewards_json = '[]'::jsonb
  );

-- 已领取任务不会重新执行接取初始化，因此需要根据修复后的模板补齐缺失目标。
-- 待提交任务按已完成目标补齐；进行中任务从 0 开始，玩家可再次触发对应服务端事件。
INSERT INTO player_quest_objective (
    player_id,
    quest_id,
    objective_id,
    event_type,
    description,
    current_value,
    target_value,
    completed,
    target_selector_json,
    guide_json
)
SELECT
    pq.player_id,
    pq.quest_id,
    (objective.value->>'objective_id')::BIGINT,
    objective.value->>'event_type',
    COALESCE(objective.value->>'description', ''),
    CASE
        WHEN pq.state = 'READY_TO_SUBMIT' THEN (objective.value->>'target')::BIGINT
        ELSE 0
    END,
    (objective.value->>'target')::BIGINT,
    pq.state = 'READY_TO_SUBMIT',
    COALESCE(objective.value->'target_selector', '{}'::jsonb),
    COALESCE(objective.value->'guide', '{}'::jsonb)
FROM player_quest AS pq
JOIN quest_template AS qt
  ON qt.quest_id = pq.quest_id
CROSS JOIN LATERAL jsonb_array_elements(qt.objectives_json) AS objective(value)
WHERE pq.quest_id IN (1101, 1102, 1103)
  AND pq.state IN ('ACCEPTED', 'READY_TO_SUBMIT')
ON CONFLICT (player_id, quest_id, objective_id) DO NOTHING;

COMMIT;

-- 回滚说明：本迁移修复的是正式任务配置和缺失的玩家目标数据，不提供自动删除回滚，
-- 避免误删迁移执行后已经产生真实进度的 player_quest_objective 记录。
