-- 079_backfill_missing_item_equipment_extra.sql
-- 部分装备模板（如 4002 新手长剑(10)）缺少 item_equipment_extra，导致 equip_slot 为空、
-- 背包不下发 enhance_preview，客户端无法打开强化材料列表。

INSERT INTO item_equipment_extra (
  item_id,
  equip_slot,
  base_hp,
  base_mana,
  base_atk,
  base_def,
  base_spd,
  career_limit,
  pet_only,
  player_only,
  extra_rule_json,
  can_enhance,
  max_enhance_level,
  set_id,
  appearance_skin_id,
  appearance_only,
  base_stats_json,
  enhance_per_level_stats_json,
  socket_count,
  allowed_gem_types_json,
  enhance_gold_cost_enabled,
  enhance_gold_base_copper,
  enhance_gold_increment_mode,
  enhance_gold_increment_fixed,
  enhance_gold_increment_percent,
  weapon_type,
  weapon_skills_json,
  enhance_per_level_weapon_skill_levels_json
)
SELECT
  idf.item_id,
  src.equip_slot,
  src.base_hp,
  src.base_mana,
  src.base_atk,
  src.base_def,
  src.base_spd,
  src.career_limit,
  src.pet_only,
  src.player_only,
  src.extra_rule_json,
  src.can_enhance,
  src.max_enhance_level,
  src.set_id,
  src.appearance_skin_id,
  src.appearance_only,
  src.base_stats_json,
  src.enhance_per_level_stats_json,
  src.socket_count,
  src.allowed_gem_types_json,
  src.enhance_gold_cost_enabled,
  src.enhance_gold_base_copper,
  src.enhance_gold_increment_mode,
  src.enhance_gold_increment_fixed,
  src.enhance_gold_increment_percent,
  src.weapon_type,
  src.weapon_skills_json,
  src.enhance_per_level_weapon_skill_levels_json
FROM item_definition AS idf
CROSS JOIN item_equipment_extra AS src
WHERE idf.item_type = 'equipment'
  AND src.item_id = 4001
  AND NOT EXISTS (
    SELECT 1
    FROM item_equipment_extra AS iee
    WHERE iee.item_id = idf.item_id
  );
