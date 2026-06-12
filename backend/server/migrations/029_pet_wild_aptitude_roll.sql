-- 029_pet_wild_aptitude_roll.sql
-- 野外捕捉宠物模板配置资质 roll 范围；玩家宠物实例持久化最终资质。

ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS hp_apt_roll_min INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS hp_apt_roll_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS atk_apt_roll_min INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS atk_apt_roll_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS def_apt_roll_min INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS def_apt_roll_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS spd_apt_roll_min INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS spd_apt_roll_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS mana_apt_roll_min INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS mana_apt_roll_max INTEGER NOT NULL DEFAULT 0;

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS hp_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS atk_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS def_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS spd_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS mana_apt INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS grant_source VARCHAR(32) NOT NULL DEFAULT 'template',
  ADD COLUMN IF NOT EXISTS capture_monster_id INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN player_pet.grant_source IS '发放来源：template=模板固定资质，wild_capture=野外捕捉 roll 资质';
COMMENT ON COLUMN player_pet.capture_monster_id IS '野外捕捉来源怪物 monster_id，非捕捉发放时为 0';

-- 将已有玩家宠物实例的资质回填为对应模板固定值
UPDATE player_pet pp
SET
  hp_apt = pd.hp_apt,
  atk_apt = pd.atk_apt,
  def_apt = pd.def_apt,
  spd_apt = pd.spd_apt,
  mana_apt = pd.mana_apt,
  grant_source = 'template'
FROM pet_definition pd
WHERE pp.pet_id = pd.pet_id
  AND pp.grant_source = 'template'
  AND pp.hp_apt = 0;

-- 野外捕捉专用宠物模板：与 monster 9001 关联
INSERT INTO pet_definition (
  pet_id,
  pet_name,
  description,
  acquire_method,
  status,
  level,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  hp_apt,
  atk_apt,
  def_apt,
  spd_apt,
  mana_apt,
  hp_apt_roll_min,
  hp_apt_roll_max,
  atk_apt_roll_min,
  atk_apt_roll_max,
  def_apt_roll_min,
  def_apt_roll_max,
  spd_apt_roll_min,
  spd_apt_roll_max,
  mana_apt_roll_min,
  mana_apt_roll_max,
  skill_ids
) VALUES (
  103,
  '野生幼犬',
  '通过捕捉野生怪物获得的草系幼犬，各项成长资质在配置范围内随机生成。',
  'wild_capture',
  1,
  1,
  1,
  22,
  22,
  12,
  9,
  8,
  9,
  10,
  10,
  10,
  10,
  10,
  8,
  14,
  8,
  13,
  8,
  12,
  7,
  12,
  6,
  11,
  '[1001, 90001]'::jsonb
)
ON CONFLICT (pet_id) DO UPDATE SET
  pet_name = EXCLUDED.pet_name,
  description = EXCLUDED.description,
  acquire_method = EXCLUDED.acquire_method,
  status = EXCLUDED.status,
  level = EXCLUDED.level,
  quality = EXCLUDED.quality,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  mana = EXCLUDED.mana,
  hp_apt = EXCLUDED.hp_apt,
  atk_apt = EXCLUDED.atk_apt,
  def_apt = EXCLUDED.def_apt,
  spd_apt = EXCLUDED.spd_apt,
  mana_apt = EXCLUDED.mana_apt,
  hp_apt_roll_min = EXCLUDED.hp_apt_roll_min,
  hp_apt_roll_max = EXCLUDED.hp_apt_roll_max,
  atk_apt_roll_min = EXCLUDED.atk_apt_roll_min,
  atk_apt_roll_max = EXCLUDED.atk_apt_roll_max,
  def_apt_roll_min = EXCLUDED.def_apt_roll_min,
  def_apt_roll_max = EXCLUDED.def_apt_roll_max,
  spd_apt_roll_min = EXCLUDED.spd_apt_roll_min,
  spd_apt_roll_max = EXCLUDED.spd_apt_roll_max,
  mana_apt_roll_min = EXCLUDED.mana_apt_roll_min,
  mana_apt_roll_max = EXCLUDED.mana_apt_roll_max,
  skill_ids = EXCLUDED.skill_ids,
  updated_at = CURRENT_TIMESTAMP;
