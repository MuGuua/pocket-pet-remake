-- 021_pet_definition_admin_fields.sql
-- 为系统宠物模板补运营后台所需字段：描述、获取方式、成长资质。

ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS acquire_method VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS hp_apt INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS atk_apt INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS def_apt INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS spd_apt INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS mana_apt INTEGER NOT NULL DEFAULT 10;

UPDATE pet_definition
SET
  description = '擅长近身撕咬的草系幼犬，适合作为新手玩家的第一只战斗宠物。',
  acquire_method = '任务奖励 / 新手引导',
  hp_apt = 12,
  atk_apt = 11,
  def_apt = 10,
  spd_apt = 9,
  mana_apt = 8,
  updated_at = CURRENT_TIMESTAMP
WHERE pet_id = 101;

UPDATE pet_definition
SET
  description = '掌控潮汐之力的灵狐，法术输出稳定，适合中后期培养。',
  acquire_method = '任务奖励',
  hp_apt = 10,
  atk_apt = 9,
  def_apt = 10,
  spd_apt = 11,
  mana_apt = 14,
  updated_at = CURRENT_TIMESTAMP
WHERE pet_id = 102;
