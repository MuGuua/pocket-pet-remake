-- 049_character_slash_visual.sql
-- 裂空斩(1101) 客户端特效资源为 character_slash，与普攻 slash 区分。

UPDATE skill_definition
SET
  animation_key = 'character_slash',
  skill_visual_id = 'character_slash',
  updated_at = CURRENT_TIMESTAMP
WHERE skill_id = 1101;
