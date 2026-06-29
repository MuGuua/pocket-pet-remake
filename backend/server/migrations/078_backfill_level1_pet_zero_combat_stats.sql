-- 078_backfill_level1_pet_zero_combat_stats.sql
-- 1 级宠物在成长公式下分配点为 0，早期发放链路会写入 hp/atk 全 0，导致战斗判定死亡。
-- 对已损坏实例回退到 pet_definition 模板基础战斗属性。

UPDATE player_pet AS pp
SET
  hp = CASE WHEN pd.hp_max > 0 THEN pd.hp_max ELSE GREATEST(pp.hp, 1) END,
  hp_max = CASE WHEN pd.hp_max > 0 THEN pd.hp_max ELSE GREATEST(pp.hp_max, 1) END,
  atk = CASE WHEN pd.atk > 0 THEN pd.atk ELSE pp.atk END,
  def = CASE WHEN pd.def > 0 THEN pd.def ELSE pp.def END,
  spd = CASE WHEN pd.spd > 0 THEN pd.spd ELSE pp.spd END,
  mana = CASE WHEN pd.mana > 0 THEN pd.mana ELSE pp.mana END,
  updated_at = CURRENT_TIMESTAMP
FROM pet_definition AS pd
WHERE pp.pet_id = pd.pet_id
  AND pp.hp_max = 0
  AND pp.level <= 1;
