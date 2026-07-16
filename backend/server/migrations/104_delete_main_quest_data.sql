-- 删除全部主线任务模板及其运行数据。
-- 本脚本不会回收已经发放到钱包、背包、宠物或属性中的历史奖励。
BEGIN;

-- 在当前事务内固定待删除的主线任务 ID，避免后续删除模板后无法定位关联数据。
CREATE TEMP TABLE main_quest_ids_to_delete (
  quest_id BIGINT PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO main_quest_ids_to_delete (quest_id)
SELECT quest_id
FROM quest_template
WHERE UPPER(quest_type) = 'MAIN';

-- 删除直接绑定主线任务的 NPC 菜单入口，避免客户端继续显示已经失效的领取或交付操作。
DELETE FROM npc_menu_entry
WHERE linked_quest_id IN (
  SELECT quest_id FROM main_quest_ids_to_delete
);

-- 删除依赖、禁止或自动接取这些主线任务的场景触发器，避免触发后引用不存在的模板。
DELETE FROM scene_entry_trigger
WHERE required_quest_id IN (SELECT quest_id FROM main_quest_ids_to_delete)
   OR forbidden_quest_id IN (SELECT quest_id FROM main_quest_ids_to_delete)
   OR effect_accept_quest_id IN (SELECT quest_id FROM main_quest_ids_to_delete);

-- 保留支线和日常任务，但从它们的前置任务数组中移除已经删除的主线 ID。
UPDATE quest_template AS template
SET pre_quest_ids = COALESCE(
  (
    SELECT jsonb_agg(item.value ORDER BY item.ordinality)
    FROM jsonb_array_elements(template.pre_quest_ids) WITH ORDINALITY AS item(value, ordinality)
    WHERE NOT EXISTS (
      SELECT 1
      FROM main_quest_ids_to_delete AS deleted
      WHERE deleted.quest_id = (item.value #>> '{}')::BIGINT
    )
  ),
  '[]'::JSONB
)
WHERE UPPER(template.quest_type) <> 'MAIN'
  AND jsonb_typeof(template.pre_quest_ids) = 'array';

-- 先删除玩家侧明细，再删除任务模板，保持清理顺序明确。
DELETE FROM player_quest_event_log
WHERE quest_id IN (
  SELECT quest_id FROM main_quest_ids_to_delete
);

DELETE FROM player_quest_objective
WHERE quest_id IN (
  SELECT quest_id FROM main_quest_ids_to_delete
);

DELETE FROM player_quest
WHERE quest_id IN (
  SELECT quest_id FROM main_quest_ids_to_delete
);

DELETE FROM quest_template
WHERE quest_id IN (
  SELECT quest_id FROM main_quest_ids_to_delete
);

COMMIT;
