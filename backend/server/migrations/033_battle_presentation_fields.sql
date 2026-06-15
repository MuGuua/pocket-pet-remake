-- 033_battle_presentation_fields.sql
-- 战斗表现层字段：宠物/怪物外观 skin_id、技能表现 skill_visual_id。

ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS skin_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN pet_definition.skin_id IS '战斗场景客户端 UnitSkin 资源 ID，对应 client/resources/battle/unit_skins/{skin_id}.tres';

ALTER TABLE monster_definition
  ADD COLUMN IF NOT EXISTS skin_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN monster_definition.skin_id IS '战斗场景客户端 UnitSkin 资源 ID';

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS skill_visual_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN skill_definition.skill_visual_id IS '战斗场景客户端 SkillVisualConfig 资源 ID；为空时客户端回退 animation_key';

UPDATE pet_definition
SET skin_id = CASE pet_id
  WHEN 101 THEN '嫩叶犬_001'
  WHEN 102 THEN '潮汐狐_001'
  ELSE skin_id
END,
updated_at = CURRENT_TIMESTAMP
WHERE pet_id IN (101, 102);

UPDATE monster_definition
SET skin_id = CASE monster_id
  WHEN 9001 THEN '史莱姆_001'
  WHEN 9002 THEN '史莱姆_001'
  ELSE skin_id
END,
updated_at = CURRENT_TIMESTAMP
WHERE monster_id IN (9001, 9002);
