-- 073_player_weapon_skill_progress.sql
-- 武器类型、武器技能流派、玩家武器技能学习进度。

ALTER TABLE item_equipment_extra
  ADD COLUMN IF NOT EXISTS weapon_type VARCHAR(16) NOT NULL DEFAULT '';

COMMENT ON COLUMN item_equipment_extra.weapon_type IS
  '武器类型：sword(剑)/spear(枪)/staff(法杖)，仅 weapon/class_weapon 槽位使用';

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS weapon_discipline VARCHAR(16) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS learn_exp_required INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS learn_exp_per_use INTEGER NOT NULL DEFAULT 1;

COMMENT ON COLUMN skill_definition.weapon_discipline IS
  '武器技能流派：sword/spear/staff，skill_category=weapon 时使用';
COMMENT ON COLUMN skill_definition.learn_exp_required IS
  '武器技能学会所需累计经验；0 表示不可通过战斗学习';
COMMENT ON COLUMN skill_definition.learn_exp_per_use IS
  '战斗中每次成功使用该武器技能增加的经验';

CREATE TABLE IF NOT EXISTS player_skill_progress (
  player_id BIGINT NOT NULL,
  skill_id INTEGER NOT NULL,
  skill_exp INTEGER NOT NULL DEFAULT 0,
  skill_level INTEGER NOT NULL DEFAULT 1,
  is_learned BOOLEAN NOT NULL DEFAULT FALSE,
  learned_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_player_skill_progress_player_learned
  ON player_skill_progress (player_id, is_learned);

CREATE TRIGGER set_player_skill_progress_updated_at
BEFORE UPDATE ON player_skill_progress
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
