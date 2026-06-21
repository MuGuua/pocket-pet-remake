-- 062_skill_pocket_damage_fields.sql
-- 口袋伤害新表：技能倍数与技能附加爆伤，供 battle/formula 分子链路读取。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS skill_mult INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS skill_crit_add INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN skill_definition.skill_mult IS '口袋伤害表 SkillMult；>0 时优先于 attack_pct/100 作为面板倍率';
COMMENT ON COLUMN skill_definition.skill_crit_add IS '口袋伤害表 SkillCritAdd；与爆伤、抗爆共同组成分子爆伤链';
