-- 072_equipment_weapon_skills.sql
-- 为人物装备模板增加武器附加技能配置，以及强化等级对武器技能等级的成长配置。

ALTER TABLE item_equipment_extra
  ADD COLUMN IF NOT EXISTS weapon_skills_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS enhance_per_level_weapon_skill_levels_json JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN item_equipment_extra.weapon_skills_json IS
  '武器附加技能列表 JSON：[{"skill_id":1201,"base_level":1}]，仅 weapon/class_weapon 槽位使用';
COMMENT ON COLUMN item_equipment_extra.enhance_per_level_weapon_skill_levels_json IS
  '武器技能等级强化成长 JSON：{"1201":1} 表示每强化 1 级，对应 skill_id 的技能等级 +1';
