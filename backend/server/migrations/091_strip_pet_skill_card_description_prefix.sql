-- 删除宠物公共技能描述中的技能卡使用说明前缀，只保留真正的技能效果说明。
-- 数据来源参考 docs/pocket_pet_public_skill_definition.csv；正式执行由用户在数据库迁移流程中触发。
UPDATE skill_definition
SET
  description = replace(description, '对宠物使用此技能卡后，宠物得到这个技能：', ''),
  updated_at = now()
WHERE skill_category = 'pet'
  AND description LIKE '%对宠物使用此技能卡后，宠物得到这个技能：%';
