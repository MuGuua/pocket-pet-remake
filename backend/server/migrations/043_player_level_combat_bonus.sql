-- 043_player_level_combat_bonus.sql
-- 等级经验表增加升级战斗属性加成；完成该等级升级时累加到玩家 base_* 裸装基础值。
-- player_level_config.level 仍表示「处于该等级时升到下一级」的配置行。

ALTER TABLE player_level_config
  ADD COLUMN IF NOT EXISTS bonus_atk INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS bonus_hp_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS bonus_spd INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS bonus_mana INTEGER NOT NULL DEFAULT 0;

ALTER TABLE player_level_config
  DROP CONSTRAINT IF EXISTS ck_player_level_config_bonus_atk;

ALTER TABLE player_level_config
  ADD CONSTRAINT ck_player_level_config_bonus_atk CHECK (bonus_atk >= 0);

ALTER TABLE player_level_config
  DROP CONSTRAINT IF EXISTS ck_player_level_config_bonus_hp_max;

ALTER TABLE player_level_config
  ADD CONSTRAINT ck_player_level_config_bonus_hp_max CHECK (bonus_hp_max >= 0);

ALTER TABLE player_level_config
  DROP CONSTRAINT IF EXISTS ck_player_level_config_bonus_spd;

ALTER TABLE player_level_config
  ADD CONSTRAINT ck_player_level_config_bonus_spd CHECK (bonus_spd >= 0);

ALTER TABLE player_level_config
  DROP CONSTRAINT IF EXISTS ck_player_level_config_bonus_mana;

ALTER TABLE player_level_config
  ADD CONSTRAINT ck_player_level_config_bonus_mana CHECK (bonus_mana >= 0);

-- 1~99 级默认每升一级：攻击+7、最大生命+38、速度+2、法力+1；100 级无下一级，加成置 0。
UPDATE player_level_config
SET bonus_atk = CASE WHEN level >= 100 THEN 0 ELSE 7 END,
    bonus_hp_max = CASE WHEN level >= 100 THEN 0 ELSE 38 END,
    bonus_spd = CASE WHEN level >= 100 THEN 0 ELSE 2 END,
    bonus_mana = CASE WHEN level >= 100 THEN 0 ELSE 1 END,
    updated_at = CURRENT_TIMESTAMP;
