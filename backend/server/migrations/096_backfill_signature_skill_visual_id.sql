-- 为已存在的幻影闪击技能补齐客户端表现资源标识。
-- 仅处理空值，避免覆盖运营后台已经显式配置的其他表现资源。
UPDATE skill_definition
SET skill_visual_id = 'pet_圣技_幻影闪击',
    updated_at = NOW()
WHERE skill_id = 20191
  AND BTRIM(COALESCE(skill_visual_id, '')) = '';
