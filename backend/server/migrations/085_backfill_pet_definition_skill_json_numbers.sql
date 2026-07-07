-- 085_backfill_pet_definition_skill_json_numbers.sql
-- 历史 pet_definition 技能 JSON 字段里混入了字符串数字数组（如 ["101","102"]），
-- 后台详情接口在按 []uint32 读取时会触发 JSON 反序列化失败。
-- 本脚本把 skill_ids / innate_skill_ids / normal_skill_ids 中“可安全转成数字”的字符串项
-- 回填为真正的 JSON 数字数组，避免后台详情与后续链路继续读取到脏格式。

UPDATE pet_definition AS pd
SET skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pd.skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pd.skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pd.skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pd.skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );

UPDATE pet_definition AS pd
SET innate_skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pd.innate_skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pd.innate_skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pd.innate_skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pd.innate_skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );

UPDATE pet_definition AS pd
SET normal_skill_ids = COALESCE(
  (
    SELECT jsonb_agg((trim(items.skill_value))::bigint ORDER BY items.ordinality)
    FROM jsonb_array_elements_text(COALESCE(pd.normal_skill_ids, '[]'::jsonb)) WITH ORDINALITY AS items(skill_value, ordinality)
  ),
  '[]'::jsonb
)
WHERE jsonb_typeof(COALESCE(pd.normal_skill_ids, '[]'::jsonb)) = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pd.normal_skill_ids, '[]'::jsonb)) AS items(value)
    WHERE jsonb_typeof(items.value) = 'string'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(COALESCE(pd.normal_skill_ids, '[]'::jsonb)) AS items(skill_value)
    WHERE trim(items.skill_value) !~ '^[0-9]+$'
  );
