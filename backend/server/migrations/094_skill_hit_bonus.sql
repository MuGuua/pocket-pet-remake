-- 094_skill_hit_bonus.sql
-- 新增技能释放期间的一次性命中加成，参与本次技能命中/闪避判定，不写回角色或宠物基础属性。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS skill_hit_bonus INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN skill_definition.skill_hit_bonus IS '技能本次释放的命中加成；只参与本次命中/闪避判定，不改变施法者永久命中属性';
