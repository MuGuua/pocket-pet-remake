-- 088_backfill_player_pet_skill_json_numbers.sql
-- 历史 player_pet 技能 JSON 字段里混入了字符串数字数组（如 ["20001","20002"]），
-- 后台玩家宠物列表/详情/编队接口在按 []uint32 读取时会触发 JSON 反序列化失败。
-- 本脚本把 skill_ids / innate_skill_ids / normal_skill_ids 中“可安全转成数字”的字符串项
-- 回填为真正的 JSON 数字数组，避免后台继续依赖运行时代码兼容历史脏格式。

UPDATE player_pet AS pp
SET skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pp.skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pp.skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pp.skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pp.skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );

UPDATE player_pet AS pp
SET innate_skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pp.innate_skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pp.innate_skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pp.innate_skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pp.innate_skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );

UPDATE player_pet AS pp
SET normal_skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pp.normal_skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pp.normal_skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pp.normal_skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pp.normal_skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );
