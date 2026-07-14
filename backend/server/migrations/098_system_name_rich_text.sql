-- 系统配置名称允许保存 Godot BBCode；玩家名与玩家自定义宠物名保持原有纯文本字段。
-- 改为 TEXT 避免 [color]、[b]、[i]、[u] 标签占用可见名称的原长度上限。

ALTER TABLE item_definition ALTER COLUMN item_name TYPE TEXT;
ALTER TABLE pet_definition ALTER COLUMN pet_name TYPE TEXT;
ALTER TABLE skill_definition ALTER COLUMN skill_name TYPE TEXT;
ALTER TABLE monster_definition ALTER COLUMN monster_name TYPE TEXT;
ALTER TABLE quest_template ALTER COLUMN name TYPE TEXT;
ALTER TABLE equipment_set_definition ALTER COLUMN set_name TYPE TEXT;
ALTER TABLE equipment_set_effect ALTER COLUMN effect_name TYPE TEXT;
ALTER TABLE monster_encounter ALTER COLUMN encounter_name TYPE TEXT;
ALTER TABLE scene_wild_encounter ALTER COLUMN encounter_name TYPE TEXT;

COMMENT ON COLUMN item_definition.item_name IS '物品系统名称，支持 Godot BBCode 富文本';
COMMENT ON COLUMN pet_definition.pet_name IS '宠物系统名称，支持 Godot BBCode 富文本';
COMMENT ON COLUMN skill_definition.skill_name IS '技能系统名称，支持 Godot BBCode 富文本';
COMMENT ON COLUMN monster_definition.monster_name IS '怪物系统名称，支持 Godot BBCode 富文本';
